package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

/*
SUMMARY OF FUNCTIONS USED
- open_db: Opens connection to database
- read_rows: Reads the needed information from each row
- check_name: Checks whether the name is correct and eliminates adjectives
- compare_name: Compares name with the given name in database
- insert_code: Insert the new information into the database
- separate_str: Separate the string into words to get rid of adjectives
- check_adj: Check if any of the word is an adjective
- fuzzy_search: Autocorrect small mistakes or differences in the name
*/

const (
	host     = HOST
	port     = PORT
	user     = USER
	password = PASSWORD
	dbname   = DBNAME
)

func main() {
	db, err := open_db()

	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	defer db.Close()

	rows, err := db.Query("SELECT DISTINCT area1 FROM transport_data ORDER BY area1")

	if err != nil {
		log.Printf("Error querying rows: %v", err)
		return
	}

	source := 1
	read_rows(db, rows, source)
	rows.Close()

	rows, err = db.Query("SELECT DISTINCT area2 FROM transport_data ORDER BY area2")

	if err != nil {
		log.Printf("Error querying rows: %v", err)
		return
	}

	source = 0
	read_rows(db, rows, source)
	rows.Close()
}

func open_db() (*sql.DB, error) {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)

	return db, err
}

func read_rows(db *sql.DB, rows *sql.Rows, source int) {
	area_name := ""

	for rows.Next() {
		err := rows.Scan(&area_name)

		if err != nil {
			log.Printf("Error scanning rows: %v", err)
		}

		check_name(db, area_name, source)
	}
}

func check_name(db *sql.DB, area_name string, s_d int) {
	correct_name := area_name
	count := compare_name(db, area_name)

	if count > 0 {
		insert_code(db, area_name, correct_name, s_d)

		return
	}

	correct_name = separate_str(area_name)
	count = compare_name(db, correct_name)

	if count == 1 {
		insert_code(db, area_name, correct_name, s_d)
	} else if count == 0 {
		correct_name = fuzzy_search(db, area_name)
		insert_code(db, area_name, correct_name, s_d)
	}
}

func compare_name(db *sql.DB, area_name string) int {
	count := 0

	err := db.QueryRow("SELECT COUNT(*) FROM area_info WHERE province = $1 OR city = $1 OR district = $1", area_name).Scan(&count)

	if err != nil {
		log.Printf("Error finding name in database: %v", err)
	}

	return count
}

func insert_code(db *sql.DB, area_name string, correct_name string, s_d int) {
	area_id := ""
	num_hyph := 0
	err := db.QueryRow("SELECT area_id FROM area_info WHERE province = $1 OR city = $1 OR district = $1", correct_name).Scan(&area_id)

	if area_id == "" {
		area_id = "NOT IN DATABASE"

		if s_d == 1 {
			_, err = db.Exec("UPDATE transport_data SET area_id1 = $1 WHERE area1 = $2", area_id, area_name)
		} else if s_d == 0 {
			_, err = db.Exec("UPDATE transport_data SET area_id2 = $1 WHERE area2 = $2", area_id, area_name)
		}

	} else {
		if s_d == 1 {
			_, err = db.Exec("UPDATE transport_data SET area_id1 = $1 WHERE area1 = $2", area_id, area_name)
		} else if s_d == 0 {
			_, err = db.Exec("UPDATE transport_data SET area_id2 = $1 WHERE area2 = $2", area_id, area_name)
		}

		num_hyph = strings.Count(area_id, "-")

		area_id = strings.Replace(area_id, "-", " ", -1)
		sep_area_id := strings.Fields(area_id)

		area_ids, err := strconv.Atoi(sep_area_id[0])

		if err != nil {
			log.Printf("Error converting string to integer: %v", err)
		}

		if s_d == 1 && num_hyph >= 0 {
			_, err = db.Exec("UPDATE transport_data SET prov_id1 = $1 WHERE area1 = $2", area_ids, area_name)
		} else if s_d == 0 && num_hyph >= 0 {
			_, err = db.Exec("UPDATE transport_data SET prov_id2 = $1 WHERE area2 = $2", area_ids, area_name)
		}

		if err != nil {
			log.Printf("Error updating database: %v", err)
		}

		if s_d == 1 && num_hyph >= 1 {
			area_ids, err = strconv.Atoi(sep_area_id[1])

			if err != nil {
				log.Printf("Error converting string to integer: %v", err)
			}

			_, err = db.Exec("UPDATE transport_data SET city_id1 = $1 WHERE area1 = $2", area_ids, area_name)
		} else if s_d == 0 && num_hyph >= 1 {
			area_ids, err = strconv.Atoi(sep_area_id[1])

			if err != nil {
				log.Printf("Error converting string to integer: %v", err)
			}

			_, err = db.Exec("UPDATE transport_data SET city_id2 = $1 WHERE area2 = $2", area_ids, area_name)
		}

		if err != nil {
			log.Printf("Error updating database: %v", err)
		}

		if s_d == 1 && num_hyph == 2 {
			area_ids, err = strconv.Atoi(sep_area_id[2])

			if err != nil {
				log.Printf("Error converting string to integer: %v", err)
			}

			_, err = db.Exec("UPDATE transport_data SET dist_id1 = $1 WHERE area1 = $2", area_ids, area_name)
		} else if s_d == 0 && num_hyph == 2 {
			area_ids, err = strconv.Atoi(sep_area_id[2])

			if err != nil {
				log.Printf("Error converting string to integer: %v", err)
			}

			_, err = db.Exec("UPDATE transport_data SET dist_id2 = $1 WHERE area2 = $2", area_ids, area_name)
		}

		if err != nil {
			log.Printf("Error updating database: %v", err)
		}
	}

	if err != nil {
		log.Printf("Error updating database: %v", err)
	}
}

func separate_str(area_name string) string {
	correct_name := ""
	separated_str := strings.Fields(area_name)
	num_words := len(separated_str)
	adj_found := 0

	if num_words > 1 {
		adj_found = check_adj(separated_str)
	}

	if adj_found == 1 {
		separated_str = separated_str[1:]
		correct_name = strings.Join(separated_str, " ")
	}

	return correct_name
}

func check_adj(separated_str []string) int {
	adj_found := 0
	adj := [12]string{"Kota", "Kabupaten", "Kecamatan", "Kec.", "Kec", "Kelurahan", "Kel.", "Kel", "DKI", "Adm.", "Kepulauan", "Kep."}
	i := 0

	for i < 12 {
		if strings.EqualFold(separated_str[0], adj[i]) {
			adj_found = 1
		}

		i++
	}

	return adj_found
}

func fuzzy_search(db *sql.DB, area_name string) string {
	correct_name := ""

	rows, err := db.Query("SELECT province FROM area_info WHERE city IS NULL")

	if err != nil {
		log.Printf("Error doing fuzzy search: %v", err)
	}

	curr_area := ""
	closest_area := ""
	min_dist := 100

	for rows.Next() {
		err = rows.Scan(&curr_area)

		if err != nil {
			log.Printf("Error scanning in fuzzy search: %v", err)
			continue
		}

		dist := fuzzy.LevenshteinDistance(area_name, curr_area)

		if dist < min_dist {
			closest_area = curr_area
			min_dist = dist
		}
	}

	rows.Close()

	rows, err = db.Query("SELECT city FROM area_info WHERE district IS NULL AND city IS NOT NULL")

	if err != nil {
		log.Printf("Error doing fuzzy search: %v", err)
	}

	for rows.Next() {
		err = rows.Scan(&curr_area)

		if err != nil {
			log.Printf("Error scanning in fuzzy search: %v", err)
			continue
		}

		dist := fuzzy.LevenshteinDistance(area_name, curr_area)

		if dist < min_dist {
			closest_area = curr_area
			min_dist = dist
		}
	}

	rows, err = db.Query("SELECT district FROM area_info WHERE district IS NOT NULL")

	if err != nil {
		log.Printf("Error doing fuzzy search: %v", err)
	}

	for rows.Next() {
		err = rows.Scan(&curr_area)

		if err != nil {
			log.Printf("Error scanning in fuzzy search: %v", err)
			continue
		}

		dist := fuzzy.LevenshteinDistance(area_name, curr_area)

		if dist < min_dist {
			closest_area = curr_area
			min_dist = dist
		}
	}

	if min_dist < 3 {
		correct_name = closest_area
	}

	return correct_name
}
