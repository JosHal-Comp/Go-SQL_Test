package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "postgres"
)

type TaskList struct {
	code  int
	task  string
	exist bool
	next  *TaskList
}

type GetList struct {
	Code  int    `json:"code"`
	Task  string `json:"task"`
	Exist bool   `json:"exist"`
}

func main() {
	List := new(TaskList)
	List = List.read_database()

	fmt.Printf("Commands for task list:\n1-Insert Task\n2-Delete Task\n3-API\n4-Exit Program\n")
	List.list()
	command := 0

	for command != 3 && command != 4 {
		fmt.Printf("Enter command: ")
		fmt.Scanf("%d\n", &command)

		if command == 1 {
			List = List.insert()
			List.list()
		} else if command == 2 {
			List = List.delete()
			List.list()
		} else if command == 3 || command == 4 {
			break
		} else {
			fmt.Printf("\nCommands for task list:\n1-Insert Task\n2-Delete Task\n3-API\n4-Exit Program\n")
			List.list()
		}
	}

	if command == 3 {
		handle_requests()
	}

	List.close_database()
}

func (curr *TaskList) insert() *TaskList {
	fmt.Printf("What task would you like to add: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	curr_task := scanner.Text()

	curr_pos := curr
	for curr_pos != nil {
		if curr_pos.task == curr_task && curr_pos.exist {
			fmt.Println("\nThe task already exists")
			return curr
		} else if curr_pos.task == curr_task && !curr_pos.exist {
			curr_pos.exist = true
			return curr
		}

		curr_pos = curr_pos.next
	}

	if curr == nil || curr.code == 0 {
		curr = &TaskList{1, curr_task, true, nil}
	} else {
		pos := 2
		curr_pos = curr
		for curr_pos.next != nil {
			curr_pos = curr_pos.next
			pos++
		}
		curr_pos.next = &TaskList{pos, curr_task, true, nil}
	}

	return curr
}

func (curr *TaskList) list() {
	if curr.code == 0 {
		fmt.Println("\nYou currently have no tasks.")
	} else {
		fmt.Printf("\nTask List\n")
		fmt.Printf("Code Task\n")
		for curr != nil {
			if curr.exist {
				fmt.Printf("%02d   %s\n", curr.code, curr.task)
			}
			curr = curr.next
		}
	}

	fmt.Printf("\n")
}

func (curr *TaskList) delete() *TaskList {
	fmt.Printf("What task would you like to delete (index): ")
	delete_code := 0
	fmt.Scanf("%d\n", &delete_code)

	if curr != nil {
		curr_pos := curr
		for curr_pos != nil {
			if curr_pos.code == delete_code {
				curr_pos.exist = false
			}

			curr_pos = curr_pos.next
		}
	}

	return curr
}

func (curr *TaskList) read_database() *TaskList {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	rows, err := db.Query("SELECT * FROM Tasks ORDER BY code")

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	code_num := 0
	task_name := ""
	existence := false
	var head *TaskList
	var curr_pos *TaskList

	for rows.Next() {
		err = rows.Scan(&code_num, &task_name, &existence)
		if err != nil {
			panic(err)
		}

		new_node := &TaskList{code_num, task_name, existence, nil}

		if head == nil {
			head = new_node
			curr_pos = head
		} else {
			curr_pos.next = new_node
			curr_pos = new_node
		}
	}

	curr = head

	return curr
}

func (curr *TaskList) close_database() {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	curr_pos := curr
	count := 0

	for curr_pos != nil {
		err = db.QueryRow("SELECT COUNT(*) FROM Tasks WHERE code = $1", curr_pos.code).Scan(&count)

		if err != nil {
			panic(err)
		}

		if count == 1 {
			_, err := db.Exec("UPDATE Tasks SET exist = $1 WHERE code = $2", curr_pos.exist, curr_pos.code)
			if err != nil {
				panic(err)
			}
		} else if count == 0 {
			_, err := db.Exec("INSERT INTO Tasks (task, exist) VALUES ($1, $2)", curr_pos.task, curr_pos.exist)
			if err != nil {
				panic(err)
			}
		}

		curr_pos = curr_pos.next
	}
}

func get_tasks(w http.ResponseWriter, r *http.Request) {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	rows, err := db.Query("SELECT * FROM Tasks ORDER BY code")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var tasks []GetList

	for rows.Next() {
		var task GetList
		err := rows.Scan(&task.Code, &task.Task, &task.Exist)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func create_task(w http.ResponseWriter, r *http.Request) {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	var insert_task GetList
	params := mux.Vars(r)
	insert_task.Task = params["task"]

	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	count := 0

	err = db.QueryRow("SELECT COUNT(*) FROM Tasks WHERE task = $1", insert_task.Task).Scan(&count)

	if err != nil {
		panic(err)
	}

	deleted := 0

	if count == 1 {
		existence := true
		err = db.QueryRow("SELECT exist FROM Tasks WHERE task = $1", insert_task.Task).Scan(&existence)

		if err != nil {
			panic(err)
		}

		if !existence {
			deleted = 1
		}
	} else {
		_, err = db.Exec("INSERT INTO Tasks (task, exist) VALUES ($1, $2)", insert_task.Task, true)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if deleted == 1 {
		_, err = db.Exec("UPDATE Tasks SET exist = $1 WHERE task = $2", true, insert_task.Task)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func delete_task(w http.ResponseWriter, r *http.Request) {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])

	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	count := 0

	err = db.QueryRow("SELECT COUNT(*) FROM Tasks WHERE code = $1", id).Scan(&count)

	if err != nil {
		panic(err)
	}

	if count == 1 {
		_, err = db.Exec("UPDATE Tasks SET exist = $1 WHERE code = $2", false, id)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handle_requests() {
	router := mux.NewRouter()
	router.HandleFunc("/tasks", get_tasks).Methods("GET")
	router.HandleFunc("/tasks/{task}", create_task).Methods("POST")
	router.HandleFunc("/tasks/{id}", delete_task).Methods("DELETE")

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}
