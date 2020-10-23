package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/jmoiron/sqlx"
	"github.com/xbsoftware/querysql"
	"log"
	"net/http"
	"strings"
	"time"
)

func dataAPI(r *chi.Mux, db *sqlx.DB) {

	r.Get("/api/fields/{name}/suggest", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		parts := strings.Split(name, ".")
		table := parts[0]
		field := parts[1]

		f, err := getFieldInfo(table, field)
		if err != nil {
			format.Text(w, 500, err.Error())
			return
		}

		sql := fmt.Sprintf("select distinct %s from %s ORDER BY %s ASC", field, table, field)
		if f.Type == StringField {
			out := make([]string, 0)
			err := db.Select(&out, sql)
			if err != nil {
				fmt.Printf("%+v", err)
			}
			format.JSON(w, 200, out)
			return
		}

		if f.Type == NumberField {
			out := make([]float64, 0)
			err := db.Select(&out, sql)
			if err != nil {
				log.Printf("%+v", err)
			}
			format.JSON(w, 200, out)
			return
		}

		if f.Type == DateField {
			out := make([]time.Time, 0)
			err := db.Select(&out, sql)
			if err != nil {
				log.Printf("%+v", err)
			}
			format.JSON(w, 200, out)
			return
		}

		format.JSON(w, 200, []string{})
	})

	// options
	r.Get("/api/fields/{name}/options", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		parts := strings.Split(name, ".")
		table := parts[0]
		field := parts[1]


		f, err := getFieldInfo(table, field)
		if err != nil {
			format.Text(w, 500, err.Error())
			return
		}

		if f.Type == PickListField {
			format.JSON(w, 200, picks[f.Ref])
			return
		}

		if f.Type != ReferenceField {
			format.JSON(w, 200, []Pick{})
			return
		}

		from := pull[f.Ref]
		list := []Pick{}
		sql := fmt.Sprintf("SELECT `%s` as id,`%s` as value FROM `%s`", from.Key, from.Label, f.Ref)
		fmt.Println(sql)

		err = db.Select(&list, sql)
		if err != nil {
			format.Text(w, 500, err.Error())
			return
		}

		format.JSON(w, 200, list)
	})

	r.Post("/api/objects/{id}/data", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := chi.URLParam(r, "id")
		query := []byte(r.Form.Get("query"))
		joins := []byte(r.Form.Get("joins"))
		columns := []byte(r.Form.Get("columns"))
		limit := r.Form.Get("limit")

		var err error

		var filter = querysql.Filter{}
		if len(query) > 0 {
			filter, err = querysql.FromJSON(query)
			if err != nil {
				format.Text(w, 500, err.Error())
				return
			}
		}

		var joinsData = make([]Join, 0)
		if len(joins) > 0 {
			err = json.Unmarshal(joins, &joinsData)
			if err != nil {
				format.Text(w, 500, err.Error())
				return
			}
		}

		var colsData = make([]string, 0)
		if len(columns) > 0 {
			err = json.Unmarshal(columns, &colsData)
			if err != nil {
				format.Text(w, 500, err.Error())
				return
			}
		}

		querySQL, data, err := querysql.GetSQL(filter, nil)
		if err != nil {
			format.Text(w, 500, err.Error())
			return
		}

		sql := SelectSQL(colsData) + FromSQL(id, joinsData)
		if querySQL != "" {
			sql += " where " + querySQL
		}
		if limit != "" {
			sql += " limit " + limit
		}

		fmt.Println(sql)

		rows, err := db.Queryx(sql, data...)
		if err != nil {
			format.Text(w, 500, err.Error())
			return
		}

		t := make([]RawData, 0)
		for rows.Next() {
			res := make(map[string]interface{})
			err = rows.MapScan(res)
			if err != nil {
				format.Text(w, 500, err.Error())
				return
			}

			bytesToString(res)
			t = append(t, res)
		}

		format.JSON(w, 200, t)
	})

}
