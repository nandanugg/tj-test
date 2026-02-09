package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/nandanugg/tj-test/module/core/domain"
)

func TestInsert_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ts := time.Unix(1715003456, 0)
	mock.ExpectExec(`INSERT INTO vehicle_locations`).
		WithArgs("B1234XYZ", -6.2088, 106.8456, ts).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewLocationRepo(db)
	err = repo.Insert(context.Background(), &domain.VehicleLocation{
		VehicleID: "B1234XYZ",
		Location:  domain.Location{Lat: -6.2088, Lon: 106.8456, Timestamp: ts},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInsert_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ts := time.Unix(1715003456, 0)
	mock.ExpectExec(`INSERT INTO vehicle_locations`).
		WithArgs("B1234XYZ", -6.2088, 106.8456, ts).
		WillReturnError(sqlmock.ErrCancelled)

	repo := NewLocationRepo(db)
	err = repo.Insert(context.Background(), &domain.VehicleLocation{
		VehicleID: "B1234XYZ",
		Location:  domain.Location{Lat: -6.2088, Lon: 106.8456, Timestamp: ts},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLatest_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ts := time.Unix(1715003456, 0)
	rows := sqlmock.NewRows([]string{"vehicle_id", "latitude", "longitude", "timestamp"}).
		AddRow("B1234XYZ", -6.2088, 106.8456, ts)

	mock.ExpectQuery(`SELECT vehicle_id, latitude, longitude, timestamp FROM vehicle_locations WHERE vehicle_id = (.+) ORDER BY timestamp DESC LIMIT 1`).
		WithArgs("B1234XYZ").
		WillReturnRows(rows)

	repo := NewLocationRepo(db)
	vl, err := repo.GetLatest(context.Background(), "B1234XYZ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vl.VehicleID != "B1234XYZ" {
		t.Errorf("expected B1234XYZ, got %s", vl.VehicleID)
	}
	if vl.Location.Lat != -6.2088 {
		t.Errorf("expected -6.2088, got %f", vl.Location.Lat)
	}
	if !vl.Location.Timestamp.Equal(ts) {
		t.Errorf("expected %v, got %v", ts, vl.Location.Timestamp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetLatest_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"vehicle_id", "latitude", "longitude", "timestamp"})
	mock.ExpectQuery(`SELECT vehicle_id, latitude, longitude, timestamp FROM vehicle_locations WHERE vehicle_id = (.+)`).
		WithArgs("UNKNOWN").
		WillReturnRows(rows)

	repo := NewLocationRepo(db)
	_, err = repo.GetLatest(context.Background(), "UNKNOWN")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetHistory_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ts1 := time.Unix(1715000000, 0)
	ts2 := time.Unix(1715005000, 0)
	start := time.Unix(1715000000, 0)
	end := time.Unix(1715009999, 0)

	rows := sqlmock.NewRows([]string{"vehicle_id", "latitude", "longitude", "timestamp"}).
		AddRow("B1234XYZ", -6.2, 106.8, ts1).
		AddRow("B1234XYZ", -6.3, 106.9, ts2)

	mock.ExpectQuery(`SELECT vehicle_id, latitude, longitude, timestamp FROM vehicle_locations WHERE vehicle_id = (.+) AND timestamp >= (.+) AND timestamp <= (.+) ORDER BY timestamp ASC`).
		WithArgs("B1234XYZ", start, end).
		WillReturnRows(rows)

	repo := NewLocationRepo(db)
	results, err := repo.GetHistory(context.Background(), &domain.HistoryQuery{
		VehicleID: "B1234XYZ",
		Start:     start,
		End:       end,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Location.Lat != -6.2 {
		t.Errorf("expected -6.2, got %f", results[0].Location.Lat)
	}
	if results[1].Location.Lat != -6.3 {
		t.Errorf("expected -6.3, got %f", results[1].Location.Lat)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetHistory_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	start := time.Unix(1715000000, 0)
	end := time.Unix(1715009999, 0)
	rows := sqlmock.NewRows([]string{"vehicle_id", "latitude", "longitude", "timestamp"})

	mock.ExpectQuery(`SELECT vehicle_id, latitude, longitude, timestamp FROM vehicle_locations`).
		WithArgs("B1234XYZ", start, end).
		WillReturnRows(rows)

	repo := NewLocationRepo(db)
	results, err := repo.GetHistory(context.Background(), &domain.HistoryQuery{
		VehicleID: "B1234XYZ",
		Start:     start,
		End:       end,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestGetHistory_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	start := time.Unix(1715000000, 0)
	end := time.Unix(1715009999, 0)

	mock.ExpectQuery(`SELECT vehicle_id, latitude, longitude, timestamp FROM vehicle_locations`).
		WithArgs("B1234XYZ", start, end).
		WillReturnError(sqlmock.ErrCancelled)

	repo := NewLocationRepo(db)
	_, err = repo.GetHistory(context.Background(), &domain.HistoryQuery{
		VehicleID: "B1234XYZ",
		Start:     start,
		End:       end,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetAllVehicles_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"vehicle_id"}).
		AddRow("B1234XYZ").
		AddRow("B5678ABC")

	mock.ExpectQuery(`SELECT DISTINCT vehicle_id FROM vehicle_locations`).
		WillReturnRows(rows)

	repo := NewLocationRepo(db)
	results, err := repo.GetAllVehicles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 vehicles, got %d", len(results))
	}
	if results[0].VehicleID != "B1234XYZ" {
		t.Errorf("expected B1234XYZ, got %s", results[0].VehicleID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetAllVehicles_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"vehicle_id"})
	mock.ExpectQuery(`SELECT DISTINCT vehicle_id FROM vehicle_locations`).
		WillReturnRows(rows)

	repo := NewLocationRepo(db)
	results, err := repo.GetAllVehicles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 vehicles, got %d", len(results))
	}
}
