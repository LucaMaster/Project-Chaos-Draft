package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type Mapa struct {
	ID      int    `json:"id"`
	Titulo  string `json:"titulo"`
	Baneado bool   `json:"baneado"`
}

type Juego struct {
	ID     int    `json:"id"`
	Titulo string `json:"titulo"`
	Mapas  []Mapa `json:"mapas"`
}

var db *sql.DB

func conexionDB() {
	var err error
	db, err = sql.Open("sqlite3", "./banphase.db")
	if err != nil {
		fmt.Println("Error al abrir la DB:", err)
		return
	}

	// Crear tablas con autoincrement
	sqlJuegos := `
	CREATE TABLE IF NOT EXISTS juegos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		titulo TEXT NOT NULL
	);
	`
	sqlMapas := `
	CREATE TABLE IF NOT EXISTS mapas (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		juego_id INTEGER NOT NULL,
		titulo TEXT NOT NULL,
		baneado INTEGER DEFAULT 0,
		FOREIGN KEY(juego_id) REFERENCES juegos(id)
	);
	`

	_, err = db.Exec(sqlJuegos)
	if err != nil {
		fmt.Println("Error al crear tabla juegos:", err)
		return
	}

	_, err = db.Exec(sqlMapas)
	if err != nil {
		fmt.Println("Error al crear tabla mapas:", err)
		return
	}

	fmt.Println("Conexión a SQLite exitosa y tablas listas")
}

func getJuegos(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query("SELECT id, titulo FROM juegos")
	if err != nil {
		http.Error(w, "Error al leer juegos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var juegosResp []Juego

	for rows.Next() {
		var j Juego
		if err := rows.Scan(&j.ID, &j.Titulo); err != nil {
			continue
		}

		j.Mapas = []Mapa{}

		mapasRows, err := db.Query("SELECT id, titulo, baneado FROM mapas WHERE juego_id = ?", j.ID)
		if err == nil {
			for mapasRows.Next() {
				var m Mapa
				var baneadoInt int
				if err := mapasRows.Scan(&m.ID, &m.Titulo, &baneadoInt); err != nil {
					continue
				}
				m.Baneado = baneadoInt != 0
				j.Mapas = append(j.Mapas, m)
			}
			mapasRows.Close()
		}

		juegosResp = append(juegosResp, j)
	}

	json.NewEncoder(w).Encode(juegosResp)
}

func getMapas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	juegoID := r.URL.Query().Get("juegoId")
	var rows *sql.Rows
	var err error

	if juegoID != "" {
		rows, err = db.Query("SELECT id, titulo, baneado FROM mapas WHERE juego_id = ?", juegoID)
	} else {
		rows, err = db.Query("SELECT id, titulo, juego_id, baneado FROM mapas")
	}

	if err != nil {
		http.Error(w, "Error al leer mapas", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var mapas []Mapa
	for rows.Next() {
		var id int
		var titulo string
		var baneadoInt int
		if juegoID != "" {
			rows.Scan(&id, &titulo, &baneadoInt)
		} else {
			var juegoIDTemp int
			rows.Scan(&id, &titulo, &juegoIDTemp, &baneadoInt)
		}
		mapas = append(mapas, Mapa{
			ID:      id,
			Titulo:  titulo,
			Baneado: baneadoInt != 0,
		})
	}

	json.NewEncoder(w).Encode(mapas)
}

func addJuego(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Titulo string `json:"titulo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Titulo == "" {
		http.Error(w, "Título requerido", http.StatusBadRequest)
		return
	}

	res, err := db.Exec("INSERT INTO juegos(titulo) VALUES (?)", input.Titulo)
	if err != nil {
		http.Error(w, "Error al guardar juego", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Juego{ID: int(id), Titulo: input.Titulo, Mapas: []Mapa{}})
}

func addMapa(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var payload struct {
		JuegoID int    `json:"juegoId"`
		Titulo  string `json:"titulo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Titulo == "" || payload.JuegoID == 0 {
		http.Error(w, "Datos incompletos", http.StatusBadRequest)
		return
	}

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM juegos WHERE id = ?)", payload.JuegoID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Juego no encontrado", http.StatusNotFound)
		return
	}

	res, err := db.Exec("INSERT INTO mapas(juego_id, titulo, baneado) VALUES (?, ?, 0)", payload.JuegoID, payload.Titulo)
	if err != nil {
		http.Error(w, "Error al guardar mapa", http.StatusInternalServerError)
		return
	}

	mapaID, _ := res.LastInsertId()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Mapa{ID: int(mapaID), Titulo: payload.Titulo, Baneado: false})
}

func actualizarBaneo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var payload struct {
		MapaID  int  `json:"mapaId"`
		Baneado bool `json:"baneado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Datos inválidos", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE mapas SET baneado = ? WHERE id = ?", boolToInt(payload.Baneado), payload.MapaID)
	if err != nil {
		http.Error(w, "Error al actualizar mapa", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func main() {
	conexionDB()
	defer db.Close()

	http.HandleFunc("/api/juegos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getJuegos(w, r)
		} else if r.Method == "POST" {
			addJuego(w, r)
		}
	})

	http.HandleFunc("/api/mapas", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getMapas(w, r)
		} else if r.Method == "POST" {
			addMapa(w, r)
		}
	})

	http.HandleFunc("/api/mapas/baneo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			actualizarBaneo(w, r)
		} else {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	})

	http.Handle("/", http.FileServer(http.Dir("./public")))
	fmt.Println("Servidor corriendo en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
