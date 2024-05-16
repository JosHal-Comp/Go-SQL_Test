package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Constants used to connect to database
const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "postgres"
)

// Linked list that keeps track of all the tasks, its codes and whether it has been deleted
type TaskList struct {
	code  int
	task  string
	exist bool
	next  *TaskList
}

// Struct used for API response
type GetList struct {
	Code  int    `json:"code"`
	Task  string `json:"task"`
	Exist bool   `json:"exist"`
}

func main() {
	// Initialize task linked list and get existing values from database
	List := new(TaskList)
	List = List.read_database()

	// Prints program commands and lists existing task list
	fmt.Printf("Commands for task list:\n1-Insert Task\n2-Delete Task\n3-API\n4-Exit Program\n")
	List.list()
	command := 0

	// Loop that handles insert and delete commands by user, exits loop when API command or exit program command
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

	// Start API if API command is inputted
	if command == 3 {
		handle_requests()
	}

	// Update the database with new inserted/deleted task informations
	List.close_database()
}

func (curr *TaskList) insert() *TaskList {
	// Scans task to be added to the list
	fmt.Printf("What task would you like to add: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	curr_task := scanner.Text()

	curr_pos := curr

	// Loop cycles through task linked list to check if task already exists, or has been deleted
	// If shown in database that task existed but has been deleted (exist = false), reinsert into task list (exist = true)
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

	// Append new task to end of task linked list if it never existed
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
	// Code starts at 1 so if code = 0, no tasks in list
	// Print tasks if exist is true (exist = false means it was deleted from task list)
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
	// Scans code of task to be deleted
	fmt.Printf("What task would you like to delete (code): ")
	delete_code := 0
	fmt.Scanf("%d\n", &delete_code)

	// If the task list is not empty, cycle through linked list until task code to be deleted is found
	// delete by making exist = false
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
	// Open database
	db, err := open_db()

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	defer db.Close()

	// Query database for all tasks in order of code
	rows, err := db.Query("SELECT * FROM Tasks ORDER BY code")

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	defer rows.Close()

	// Variables used for getting the query results into the linked list
	code_num := 0
	task_name := ""
	existence := false
	var head *TaskList
	var curr_pos *TaskList

	// Loop that scans task code, task and task existence and input into linked list, then move to next node while rows exist
	for rows.Next() {
		err = rows.Scan(&code_num, &task_name, &existence)

		if err != nil {
			log.Fatalf("Error: %v", err)
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
	// Open database
	db, err := open_db()

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	defer db.Close()

	curr_pos := curr
	count := 0

	// Loop that cycles through task linked list and updates database at end of program
	for curr_pos != nil {
		// Check if current task exists already in database
		err = db.QueryRow("SELECT COUNT(*) FROM Tasks WHERE code = $1", curr_pos.code).Scan(&count)

		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		// If exist, update exist value, if not, insert task into database
		if count == 1 {
			_, err := db.Exec("UPDATE Tasks SET exist = $1 WHERE code = $2", curr_pos.exist, curr_pos.code)

			if err != nil {
				log.Fatalf("Error: %v", err)
			}
		} else if count == 0 {
			_, err := db.Exec("INSERT INTO Tasks (task, exist) VALUES ($1, $2)", curr_pos.task, curr_pos.exist)

			if err != nil {
				log.Fatalf("Error: %v", err)
			}
		}

		curr_pos = curr_pos.next
	}
}

func get_tasks(w http.ResponseWriter, r *http.Request) {
	// Open database
	db, err := open_db()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer db.Close()

	// Query database for all tasks ordered by task code
	rows, err := db.Query("SELECT * FROM Tasks ORDER BY code")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var tasks []GetList

	// While rows exist, scan task code, task and task existence into struct and append it into tasks slice
	for rows.Next() {
		var task GetList
		err := rows.Scan(&task.Code, &task.Task, &task.Exist)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tasks = append(tasks, task)
	}

	// Encode tasks slice as JSON and respond to GET request
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func create_task(w http.ResponseWriter, r *http.Request) {
	// Open database
	db, err := open_db()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer db.Close()

	// Store task to be added into task list
	var insert_task GetList
	params := mux.Vars(r)
	insert_task.Task = params["task"]

	// Check whether task exists in database
	count := 0
	err = db.QueryRow("SELECT COUNT(*) FROM Tasks WHERE task = $1", insert_task.Task).Scan(&count)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deleted := 0

	// If it exists, check its exist value (deleted or still in task list), else append new task into the database
	if count == 1 {
		existence := true
		err = db.QueryRow("SELECT exist FROM Tasks WHERE task = $1", insert_task.Task).Scan(&existence)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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

	// If task was deleted, reinsert task into task list (update exist to be true)
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
	// Open database
	db, err := open_db()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer db.Close()

	// Converting delete request from string to integer
	params := mux.Vars(r)
	code, err := strconv.Atoi(params["code"])

	if err != nil {
		http.Error(w, "Invalid task code", http.StatusBadRequest)
		return
	}

	// Check if task with input code exists
	count := 0
	err = db.QueryRow("SELECT COUNT(*) FROM Tasks WHERE code = $1", code).Scan(&count)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If task exists, delete it from printed task list by setting exist to false
	if count == 1 {
		_, err = db.Exec("UPDATE Tasks SET exist = $1 WHERE code = $2", false, code)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handle_requests() {
	// Sets up API routes and start server
	router := mux.NewRouter()
	router.HandleFunc("/tasks", get_tasks).Methods("GET")
	router.HandleFunc("/tasks/{task}", create_task).Methods("POST")
	router.HandleFunc("/tasks/{code}", delete_task).Methods("DELETE")

	err := http.ListenAndServe(":8080", router)

	if err != nil {
		log.Fatal(http.ListenAndServe(":8080", router))
	}
}

func open_db() (*sql.DB, error) {
	// Opens connection to database
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)

	return db, err
}
