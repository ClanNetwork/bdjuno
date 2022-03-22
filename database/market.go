package database

import (
	"fmt"

	dbtypes "github.com/forbole/bdjuno/v2/database/types"
	escrowtypes "github.com/ovrclk/akash/x/escrow/types/v1beta2"
	markettypes "github.com/ovrclk/akash/x/market/types/v1beta2"
)

func (db *Db) SaveGenesisLeases(leases []markettypes.Lease, height int64) error {
	for _, lease := range leases {
		leaseID, err := db.saveLeaseID(lease.LeaseID)
		if err != nil {
			return fmt.Errorf("error while storing lease ID: %s", err)
		}

		err = db.saveLease(leaseID, lease, height)
		if err != nil {
			return fmt.Errorf("error while storing lease: %s", err)
		}
	}

	return nil
}

func (db *Db) SaveLeases(responses []markettypes.QueryLeaseResponse, height int64) error {
	for _, res := range responses {
		leaseID, err := db.saveLeaseID(res.Lease.LeaseID)
		if err != nil {
			return fmt.Errorf("error while storing lease ID: %s", err)
		}

		err = db.saveLease(leaseID, res.Lease, height)
		if err != nil {
			return fmt.Errorf("error while storing lease: %s", err)
		}

		err = db.saveEscrowPayment(leaseID, res.EscrowPayment, height)
		if err != nil {
			return fmt.Errorf("error while storing escrow payment: %s", err)
		}
	}

	return nil
}

func (db *Db) saveLeaseID(leaseID markettypes.LeaseID) (int64, error) {
	fmt.Println("saving lease id")
	stmt := `
	INSERT INTO lease_id (owner_address, dseq, gseq, oseq, provider_address) 
	VALUES ($1, $2, $3, $4, $5) 
	ON CONFLICT (owner_address, dseq, gseq, oseq, provider_address) DO UPDATE SET 
		owner_address = excluded.owner_address 
	RETURNING id`

	var rowID int64
	err := db.Sql.QueryRow(stmt,
		leaseID.Owner, leaseID.DSeq, leaseID.GSeq, leaseID.OSeq, leaseID.Provider,
	).Scan(&rowID)
	if err != nil {
		return rowID, fmt.Errorf("error while storing lease ID: %s", err)
	}

	return rowID, nil
}

func (db *Db) saveLease(leaseID int64, l markettypes.Lease, height int64) error {
	fmt.Println("saving lease")

	stmt := `
	INSERT INTO lease (lease_id, lease_state, price, created_at, closed_on, height) 
	VALUES ($1, $2, $3, $4, $5, $6) 
	ON CONFLICT (lease_id) DO UPDATE 
	SET lease_state = excluded.lease_state, 
		price = excluded.price, 
		created_at = excluded.created_at,  
		closed_on = excluded.closed_on, 
		height = excluded.height 
	WHERE lease.height <= excluded.height`

	// price := dbtypes.NewDbDecCoin(l.Price)
	// priceVal, err := price.Value()
	// if err != nil {
	// 	return fmt.Errorf("error while getting price value")
	// }
	_, err := db.Sql.Exec(stmt,
		leaseID,
		l.State,
		fmt.Sprintf("(%s,%s)", l.Price.Denom, l.Price.Amount),
		l.CreatedAt,
		l.ClosedOn,
		height,
	)
	if err != nil {
		return fmt.Errorf("error while storing lease: %s", err)
	}

	return nil
}

func (db *Db) saveEscrowPayment(leaseID int64, e escrowtypes.FractionalPayment, height int64) error {
	fmt.Println("saving escrow payment")

	stmt := `
	INSERT INTO escrow_payment (lease_id, account_id, payment_id, owner_address, payment_state, rate, balance, withdrawn, height) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
	ON CONFLICT (lease_id) DO UPDATE 
	SET account_id = excluded.account_id, 
		payment_id = excluded.payment_id, 
		owner_address = excluded.owner_address, 
		payment_state = excluded.payment_state, 
		rate = excluded.rate,
		balance = excluded.balance, 
		withdrawn = excluded.withdrawn, 
		height = excluded.height 
	WHERE escrow_payment.height <= excluded.height`

	accountID := dbtypes.NewDbLeaseAccountID(e.AccountID)
	accountIDValue, err := accountID.Value()
	if err != nil {
		return fmt.Errorf("error while converting account ID to DbLeaseAccountID value: %s", err)
	}

	_, err = db.Sql.Exec(stmt,
		leaseID,
		accountIDValue,
		e.PaymentID,
		e.Owner,
		e.State,
		fmt.Sprintf("(%s,%s)", e.Rate.Denom, e.Rate.Amount),
		fmt.Sprintf("(%s,%s)", e.Balance.Denom, e.Balance.Amount),
		fmt.Sprintf("(%s,%s)", e.Withdrawn.Denom, e.Withdrawn.Amount),
		height,
	)
	if err != nil {
		return fmt.Errorf("error while storing escrow payment: %s", err)
	}

	return nil
}
