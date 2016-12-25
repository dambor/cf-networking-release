package store

import (
	"fmt"
	"net"
	"sync"
	"time"

	"silk-controller/models"
)

type Datastore interface {
	List() ([]models.Lease, error)
	Acquire(request models.NewLeaseRequest) (*models.AcquireLeaseResponse, error)
	Renew(request models.Lease) (*models.AcquireLeaseResponse, error)
	Delete(leaseID string) error
}

var NotFoundError = fmt.Errorf("lease not found")

type datastore struct {
	lock         sync.Mutex
	leases       map[string]*models.Lease
	subnetIsUsed map[string]bool
}

func NewDatastore() *datastore {
	inventory := make(map[string]bool)
	for i := 1; i < 256; i++ {
		inventory[fmt.Sprintf("10.255.%d.0/24", i)] = false
	}
	return &datastore{
		leases:       make(map[string]*models.Lease),
		subnetIsUsed: inventory,
	}
}

func (d *datastore) List() ([]models.Lease, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	slice := make([]models.Lease, len(d.leases))
	i := 0
	for _, lease := range d.leases {
		slice[i] = *lease
		i += 1
	}
	return slice, nil
}

func (d *datastore) Delete(leaseID string) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	lease, ok := d.leases[leaseID]
	if !ok {
		return NotFoundError
	}

	delete(d.leases, leaseID)
	d.subnetIsUsed[lease.Subnet] = false
	return nil
}

func (d *datastore) Renew(request models.Lease) (*models.AcquireLeaseResponse, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	lease, ok := d.leases[request.ID]
	if !ok {
		return nil, NotFoundError
	}

	if request.VtepIP != lease.VtepIP || request.Subnet != lease.Subnet {
		return nil, fmt.Errorf("data mismatch")
	}

	lease.LastRenewed = currentTime()

	return d.buildResponse(lease), nil
}

func currentTime() *time.Time {
	t := time.Now()
	return &t
}

func (d *datastore) Acquire(request models.NewLeaseRequest) (*models.AcquireLeaseResponse, error) {
	if net.ParseIP(request.VtepIP) == nil {
		return nil, fmt.Errorf("invalid vtep ip")
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	lease, found := d.tryFindExistingLease(request.VtepIP)

	if !found {
		var err error
		lease, err = d.reserveNew(request.VtepIP)
		if err != nil {
			return nil, err
		}
	}

	return d.buildResponse(lease), nil
}

func (d *datastore) tryFindExistingLease(vtepIP string) (*models.Lease, bool) {
	for _, lease := range d.leases {
		if lease.VtepIP == vtepIP {
			return lease, true
		}
	}
	return nil, false
}

func (d *datastore) reserveNew(vtepIP string) (*models.Lease, error) {
	id, err := models.NewID()
	if err != nil {
		return nil, err
	}

	subnet, err := d.findAFreeSubnet()
	if err != nil {
		return nil, err
	}

	d.subnetIsUsed[subnet] = true
	newLease := &models.Lease{
		Route: models.Route{
			Subnet: subnet,
			VtepIP: vtepIP,
		},
		ID:          id,
		LastRenewed: currentTime(),
	}
	d.leases[id] = newLease
	return newLease, nil
}

func (d *datastore) findAFreeSubnet() (string, error) {
	for subnet, isReserved := range d.subnetIsUsed {
		if !isReserved {
			return subnet, nil
		}
	}

	return "", fmt.Errorf("no more subnets available")
}

func (d *datastore) buildResponse(acquiredLease *models.Lease) *models.AcquireLeaseResponse {
	routes := []models.Route{}
	for _, lease := range d.leases {
		if lease != acquiredLease {
			routes = append(routes, lease.Route)
		}
	}

	return &models.AcquireLeaseResponse{
		Self:   *acquiredLease,
		Routes: routes,
	}
}
