package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/writer"

	_ "github.com/go-sql-driver/mysql"
)

type Tag struct {
	Tag string
}

type Header struct {
	Tag    string
	Fields []Tag
}

// USAGE is a const to have help description for CLI.
const USAGE = `mysql2parquet %s.
Usage:
	mysql2parquet [--help | --version ]
	mysql2parquet --user=foo --password=1234 --database=foo --query="SELECT * FROM users" --parquet=users.parquet
Options:
  --help       Show this help.
  --version    Print version numbers.
  --user       User for login if not current user.
  --host       Connect to host.
  --port       Port number to use for connection.
  --password   Password to use when connecting to server.
  --database   Database to use.
  --query      Execute SQL and quit.
  --parquet    File name to save SQL result in parquet format, without extension.
Tips:
  Try to use any of session variables to perform the extraction, before query:
    SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
    SET SQL_BIG_SELECTS=1;
    SET SQL_BUFFER_RESULT=1;
  Example:
    SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED; SELECT * FROM users;
`

func version() string {
	return "1.0.0"
}

func help(rc int) {
	fmt.Printf(USAGE, version())
	os.Exit(rc)
}

func main() {
	fHelp := flag.Bool("help", false, "")
	fVersion := flag.Bool("version", false, "")
	fUser := flag.String("user", "root", "")
	fHost := flag.String("host", "127.0.0.1", "")
	fPort := flag.Int("port", 3306, "")
	fPassword := flag.String("password", "", "")
	fDatabase := flag.String("database", "", "")
	fQuery := flag.String("query", "", "")
	fParquet := flag.String("parquet", "", "")

	flag.Usage = func() { help(1) }
	flag.Parse()

	switch {
	case *fVersion:
		fmt.Println(version())
		os.Exit(0)
	case *fHelp:
		help(0)
	case len(*fPassword) == 0:
		help(1)
	case len(*fDatabase) == 0:
		help(1)
	case len(*fQuery) == 0:
		help(1)
	case len(*fParquet) == 0:
		help(1)
	}

	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&multiStatements=true",
			*fUser,
			*fPassword,
			*fHost,
			*fPort,
			*fDatabase,
		),
	)
	if err != nil {
		panic(err.Error())
	}

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	rows, err := db.Query(*fQuery)
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	h := Header{}
	h.Tag = fmt.Sprintf("name=%s", *fParquet)

	types, err := rows.ColumnTypes()
	for _, s := range types {
		// saber si es unsigned?
		FieldTag := map[string]interface{}{}
		FieldTag["name"] = s.Name()

		switch s.DatabaseTypeName() {
		case "TINYINT", "INT":
			FieldTag["type"] = "INT32"
		case "BIGINT":
			FieldTag["type"] = "INT64"
		case "DECIMAL", "DOUBLE":
			FieldTag["type"] = "DOUBLE"
		case "DATE", "DATETIME", "TIMESTAMP":
			FieldTag["type"] = "BYTE_ARRAY"
			FieldTag["convertedtype"] = "UTF8"
		case "CHAR", "VARCHAR", "TEXT":
			FieldTag["type"] = "BYTE_ARRAY"
			FieldTag["convertedtype"] = "UTF8"
		default:
			FieldTag["type"] = "BYTE_ARRAY"
			FieldTag["encoding"] = "PLAIN_DICTIONARY"
			FieldTag["convertedtype"] = "UTF8"
		}

		isNull, _ := s.Nullable()
		if isNull {
			FieldTag["repetitiontype"] = "OPTIONAL"
		}

		FieldTagValue := []string{}
		for k, v := range FieldTag {
			FieldTagValue = append(FieldTagValue, fmt.Sprintf("%s=%v", k, v))
		}

		h.Fields = append(h.Fields, Tag{strings.Join(FieldTagValue, ",")})
	}

	md, _ := json.Marshal(h)

	//write
	fw, err := local.NewLocalFileWriter(fmt.Sprintf("%s.parquet", *fParquet))
	if err != nil {
		panic(err.Error())
	}

	pw, err := writer.NewJSONWriter(string(md), fw, 4)
	if err != nil {
		panic(err.Error())
	}

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		// Now do something with the data.
		// Here we just print each column as a string.
		tableData := map[string]string{}

		// var value string
		for i, col := range values {
			tableData[columns[i]] = string(col)
		}

		jsonData, err := json.Marshal(tableData)
		if err != nil {
			panic(err.Error())
		}

		// fmt.Println(string(jsonData))

		if err = pw.Write(string(jsonData)); err != nil {
			panic(err.Error())
		}
	}
	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	if err = pw.WriteStop(); err != nil {
		panic(err.Error())
	}
	fw.Close()
}
