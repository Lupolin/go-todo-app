package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type Todo struct {
	ID      int
	Title   string
	Content string
}

var db *sql.DB

func initDB() {
	var err error
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("Database connection error:", err)
	}

	// 確認連線可用
	if err = db.Ping(); err != nil {
		log.Fatal("Database unreachable:", err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS todos (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT
	);`
	if _, err := db.Exec(createTable); err != nil {
		log.Fatal("Create table error:", err)
	}
}

func main() {
	initDB()
	defer db.Close()

	// 靜態檔案
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 網頁路由
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/healthz", healthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Render 預設會使用 PORT 環境變數，不設就是 8080
	}

	log.Println("Server running at port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	rows, err := db.Query("SELECT id, title, content FROM todos")
	if err != nil {
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Content); err != nil {
			log.Println("Scan error:", err)
			continue
		}
		todos = append(todos, t)
	}

	if err := tmpl.Execute(w, todos); err != nil {
		log.Println("Template execution error:", err)
	}
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Parse form error", http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		content := r.FormValue("content")

		if title != "" {
			_, err := db.Exec("INSERT INTO todos (title, content) VALUES ($1, $2)", title, content)
			if err != nil {
				log.Println("Insert error:", err)
			}
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		_, err := db.Exec("DELETE FROM todos WHERE id = $1", id)
		if err != nil {
			log.Println("Delete error:", err)
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Database unreachable"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
