package judge

import (
	".././cookies"
	".././dao"
	".././skills"
	".././templating"
	".././users"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func stringToArray(input string) []string {
	cleaned := strings.Replace(input, " ", "", -1)
	arrInput := strings.Split(cleaned, ",")
	if arrInput[0] == "" {
		arrInput = nil
	}
	return arrInput
}

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

func EditHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		index, err := strconv.Atoi(r.URL.Path[len("/edit/"):])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		problem, err := GetProblem(index)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		templating.RenderPage(w, "editproblem", problem)
	case "POST":
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
		index, err := strconv.Atoi(r.FormValue("problem_id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p := &Problem{
			Index:        index,
			Title:        r.FormValue("title"),
			Description:  r.FormValue("description"),
			SkillID:      r.FormValue("skill"),
			Difficulty:   difficulty,
			UvaID:        r.FormValue("uva_id"),
			Input:        r.FormValue("input"),
			Output:       r.FormValue("output"),
			SampleInput:  r.FormValue("sample_input"),
			SampleOutput: r.FormValue("sample_output"),
			TimeLimit:    time_limit,
			MemoryLimit:  memory_limit,
			Tags:         stringToArray(r.FormValue("tags")),
		}
		problemQueue <- p
		editProblem(*p)
		http.Redirect(w, r, "/view/"+r.FormValue("problem_id"), http.StatusFound)
	}
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		skills, err := skills.GetAllSkills()
		if err != nil {
			templating.ErrorPage(w, 404)
			return
		}
		templating.RenderPage(w, "addproblem", skills)
	case "POST":
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
		tags := stringToArray(r.FormValue("tags"))
		p := &Problem{
			Index:        -1,
			Title:        r.FormValue("title"),
			Description:  r.FormValue("description"),
			SkillID:      r.FormValue("skill"),
			Difficulty:   difficulty,
			UvaID:        r.FormValue("uva_id"),
			Input:        r.FormValue("input"),
			Output:       r.FormValue("output"),
			SampleInput:  r.FormValue("sample_input"),
			SampleOutput: r.FormValue("sample_output"),
			TimeLimit:    time_limit,
			MemoryLimit:  memory_limit,
			Tags:         tags,
		}
		problemQueue <- p
		AddProblem(*p)
		http.Redirect(w, r, "/problems/", http.StatusFound)
	}
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		problemID, err := strconv.Atoi(r.FormValue("problem_id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		deleteProblem(problemID)
		http.Redirect(w, r, "/problems/", http.StatusFound)
	}
}

func ViewHandler(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.URL.Path[len("/view/"):])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	problem, err := GetProblem(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var userID int
	if cookies.IsLoggedIn(r) {
		userID, _ = cookies.GetUserID(r)
		users.AddViewedProblem(userID, index)
	}
	submitted, verdictData := getProblemStatistics(index)
	rate := float64(verdictData.Accepted) / float64(submitted) * 100
	if verdictData.Accepted == 0 {
		rate = 0
	}
	skill, err := getSkill(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	unlockedSkills, err := skills.GetUnlockedSkills(userID)
	data := struct {
		Problem     Problem
		Submitted   int
		Rate        float64
		VerdictData VerdictData
		Skill       skills.Skill
		Locked      bool
	}{
		problem,
		submitted,
		rate,
		verdictData,
		skill,
		!unlockedSkills[skill.ID],
	}
	templating.RenderPage(w, "viewproblem", data)
	// perhaps have a JS WARNING..
}

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	if !cookies.IsLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	index, _ := strconv.Atoi(r.URL.Path[len("/submit/"):])
	problem, err := GetProblem(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID, _ := cookies.GetUserID(r)
	unlockedSkills, err := skills.GetUnlockedSkills(userID)
	if !unlockedSkills[problem.SkillID] {
		templating.ErrorPage(w, 401)
		return
	}
	d, _ := ioutil.TempDir(DIR, "")
	ioutil.WriteFile(filepath.Join(d, "Main.java"), []byte(r.FormValue("code")), 0600)
	s := &Submission{
		UserID:       userID,
		ProblemIndex: index,
		Directory:    d,
		Verdict:      Received,
	}
	submissionID, err := addSubmission(*s, userID)
	users.UpdateAttemptedCount(userID)
	s.ID = submissionID
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go addToSubmissionQueue(s)
	http.Redirect(w, r, "/submissions/", http.StatusFound)
}

func SubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Submissions []Submission
		IsLoggedIn  bool
	}{
		getSubmissions(),
		cookies.IsLoggedIn(r),
	}
	templating.RenderPage(w, "submissions", data)
}

func SkillHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		templating.RenderPage(w, "skill", nil)
	default:
		templating.ErrorPage(w, http.StatusMethodNotAllowed)
	}
}

func SkillTreeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		templating.RenderPage(w, "skill-tree", nil)
	default:
		templating.ErrorPage(w, http.StatusMethodNotAllowed)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data := struct {
			IsLoggedIn bool
		}{
			cookies.IsLoggedIn(r),
		}
		templating.RenderPage(w, "home", data)
	default:
		templating.ErrorPage(w, http.StatusMethodNotAllowed)
	}
}
