package judge

import (
	".././cookies"
	".././dao"
	".././templating"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
)

func ProblemsHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ProblemList []Problem
		IsAdmin     bool
		IsLoggedIn  bool
	}{
		GetProblems(),
		dao.IsAdmin(r),
		cookies.IsLoggedIn(r),
	}
	templating.RenderPage(w, "viewproblems", data)
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		time_limit, err := strconv.Atoi(r.FormValue("time_limit"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		memory_limit, err := strconv.Atoi(r.FormValue("memory_limit"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		difficulty, err := strconv.Atoi(r.FormValue("difficulty"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p := &Problem{
			Index:        -1,
			Title:        r.FormValue("title"),
			Description:  r.FormValue("description"),
			Category:     r.FormValue("category"),
			Difficulty:   difficulty,
			Hint:         r.FormValue("hint"),
			Input:        r.FormValue("input"),
			Output:       r.FormValue("output"),
			SampleInput:  r.FormValue("sample_input"),
			SampleOutput: r.FormValue("sample_output"),
			TimeLimit:    time_limit,
			MemoryLimit:  memory_limit,
		}
		problemQueue <- p
		AddProblem(*p)
		http.Redirect(w, r, "/problems/", http.StatusFound)
	} else {
		templating.RenderPage(w, "addproblem", nil)
	}
}

func ViewHandler(w http.ResponseWriter, r *http.Request) {
	index, _ := strconv.Atoi(r.URL.Path[len("/view/"):])
	problem, err := GetProblem(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	templating.RenderPage(w, "viewproblem", problem)
	// perhaps have a JS WARNING..
}

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	if !cookies.IsLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	index, _ := strconv.Atoi(r.URL.Path[len("/submit/"):])
	d, _ := ioutil.TempDir(DIR, "")
	ioutil.WriteFile(filepath.Join(d, "Main.java"), []byte(r.FormValue("code")), 0600)
	s := &Submission{
		ProblemIndex: index,
		Directory:    d,
		Verdict:      Received,
	}
	userID, _ := cookies.GetUserID(r)
	submissionID, err := addSubmission(*s, userID)
	s.ID = submissionID
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	submissionQueue <- s
	http.Redirect(w, r, "/submissions/", http.StatusFound)
}

func SubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	templating.RenderPage(w, "submissions", getSubmissions())
}
