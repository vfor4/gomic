package grade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func RegisterHandlers() {
	handler := new(studentsHandler)
	http.Handle("/students", handler)
	http.Handle("/students/", handler)
}

type studentsHandler struct{}

func (sh studentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathSegments := strings.Split(r.URL.Path, "/")
	switch len(pathSegments) {
	case 2: // /students
		sh.getAll(w)
	case 3: // /students/{:id}
		id, err := strconv.Atoi(pathSegments[2])
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sh.getOne(w, id)
	case 4: // /students/{:id}/grades
		id, err := strconv.Atoi(pathSegments[2])
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.ToLower(pathSegments[3]) != "grades" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sh.addGrade(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (sh studentsHandler) getAll(w http.ResponseWriter) {
	studentsMutex.Lock()
	defer studentsMutex.Unlock()

	data, err := sh.toJSON(students)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func (sh studentsHandler) getOne(w http.ResponseWriter, id int) {
	studentsMutex.Lock()
	defer studentsMutex.Unlock()

	student, err := students.GetById(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Println(err)
		return
	}

	data, err := sh.toJSON(student)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(fmt.Errorf("Failed to serialize students: %q", err))
		return
	}
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func (studentsHandler) toJSON(obj interface{}) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	err := enc.Encode(obj)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize students: %q", err)
	}
	return b.Bytes(), nil
}

func (sh studentsHandler) addGrade(w http.ResponseWriter, r *http.Request, id int) {
	studentsMutex.Lock()
	defer studentsMutex.Unlock()

	student, err := students.GetById(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	var g Grade
	dec := json.NewDecoder(r.Body)
	err = dec.Decode(&g)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}
	student.Grades = append(student.Grades, g)

	w.WriteHeader(http.StatusCreated)
	data, err := sh.toJSON(g)
	if err != nil {
		log.Println(err)
	}
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}
