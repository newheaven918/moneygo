package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"github.com/aclindsa/moneygo/internal/config"
	"github.com/aclindsa/moneygo/internal/db"
	"github.com/aclindsa/moneygo/internal/handlers"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var server *httptest.Server

func Delete(client *http.Client, url string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(request)
}

func Put(client *http.Client, url string, contentType string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", contentType)
	return client.Do(request)
}

type TransactType interface {
	Read(string) error
}

func create(client *http.Client, input, output TransactType, urlsuffix string) error {
	obj, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return err
	}
	response, err := client.Post(server.URL+urlsuffix, "application/json", bytes.NewReader(obj))
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return err
	}

	var e handlers.Error
	err = (&e).Read(string(body))
	if err != nil {
		return err
	}
	if e.ErrorId != 0 || len(e.ErrorString) != 0 {
		return &e
	}

	err = output.Read(string(body))
	if err != nil {
		return err
	}

	return nil
}

func read(client *http.Client, output TransactType, urlsuffix string) error {
	response, err := client.Get(server.URL + urlsuffix)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return err
	}

	var e handlers.Error
	err = (&e).Read(string(body))
	if err != nil {
		return err
	}
	if e.ErrorId != 0 || len(e.ErrorString) != 0 {
		return &e
	}

	err = output.Read(string(body))
	if err != nil {
		return err
	}

	return nil
}

func update(client *http.Client, input, output TransactType, urlsuffix string) error {
	obj, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return err
	}
	response, err := Put(client, server.URL+urlsuffix, "application/json", bytes.NewReader(obj))
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return err
	}

	var e handlers.Error
	err = (&e).Read(string(body))
	if err != nil {
		return err
	}
	if e.ErrorId != 0 || len(e.ErrorString) != 0 {
		return &e
	}

	err = output.Read(string(body))
	if err != nil {
		return err
	}

	return nil
}

func remove(client *http.Client, urlsuffix string) error {
	response, err := Delete(client, server.URL+urlsuffix)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return err
	}

	var e handlers.Error
	err = (&e).Read(string(body))
	if err != nil {
		return err
	}
	if e.ErrorId != 0 || len(e.ErrorString) != 0 {
		return &e
	}

	return nil
}

func RunWith(t *testing.T, d *TestData, fn TestDataFunc) {
	testdata, err := d.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize test data: %s", err)
	}
	defer func() {
		err := testdata.Teardown()
		if err != nil {
			t.Fatal(err)
		}
	}()

	fn(t, testdata)
}

func RunTests(m *testing.M) int {
	envDbType := os.Getenv("MONEYGO_TEST_DB")
	var dbType config.DbType
	var dsn string

	switch envDbType {
	case "", "sqlite", "sqlite3":
		dbType = config.SQLite
		dsn = ":memory:"
	case "mariadb", "mysql":
		dbType = config.MySQL
		dsn = "root@127.0.0.1/moneygo_test&parseTime=true"
	case "postgres", "postgresql":
		dbType = config.Postgres
		dsn = "postgres://postgres@localhost/moneygo_test"
	default:
		log.Fatalf("Invalid value for $MONEYGO_TEST_DB: %s\n", envDbType)
	}

	if envDSN := os.Getenv("MONEYGO_TEST_DSN"); len(envDSN) > 0 {
		dsn = envDSN
	}

	dsn = db.GetDSN(dbType, dsn)
	database, err := sql.Open(dbType.String(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	dbmap, err := db.GetDbMap(database, dbType)
	if err != nil {
		log.Fatal(err)
	}

	err = dbmap.TruncateTables()
	if err != nil {
		log.Fatal(err)
	}

	server = httptest.NewTLSServer(&handlers.APIHandler{DB: dbmap})
	defer server.Close()

	return m.Run()
}

func TestMain(m *testing.M) {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	os.Exit(RunTests(m))
}
