package main

import (
	"bufio"
    "bytes"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func handler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Pass in ID and course codes", http.StatusBadRequest)
		return
	}

	matched, _ := regexp.MatchString("^([0-9A-Z]+)$", parts[1])
	if !matched {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

    courseCodes := strings.Split(parts[2], ",")
    courseCodesMap := make(map[string]bool)
    for _, courseCode := range courseCodes {
        courseCodesMap[courseCode] = true
    }

	response, err := http.Get("https://www.imperial.ac.uk/facilitiesmanagement/timetabling/mytimetable/ical/" + parts[1] + "/schedule.ics")
	if err != nil {
		http.Error(w, "Could not fetch the timetable", http.StatusServiceUnavailable)
		return
	}

	// Set the correct iCalendar MIME type
	w.Header().Set("Content-Type", "text/calendar")

	scanner := bufio.NewScanner(response.Body)
    buffering := false
    ignoring := false

    typeRegex := regexp.MustCompile("; (Lecture|Tutorial)$")
    summaryRegex := regexp.MustCompile("^SUMMARY:([a-zA-Z]{2}([0-9]{3}|-[a-zA-Z+]))")

    var buffer *bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()

        if line == "BEGIN:VEVENT" {
            buffering = true
            ignoring = false
            buffer = bytes.NewBufferString("")
        }

        if ignoring || strings.HasPrefix(line, "DESCRIPTION:") {
            continue
        }

        if strings.HasPrefix(line, "LOCATION:") {
            // Fix horrible acronyms
            line = strings.Replace(line, "HXLY", "Huxley", 1)
        } else if strings.HasPrefix(line, "SUMMARY:") {
			typeMatches := typeRegex.FindStringSubmatch(line)

            index := strings.Index(line, "\\;")
            if index != -1 {
                line = line[0:index]
            }

			// Indicate whether this was a lecture or a tutorial
			if len(typeMatches) > 0 {
				line += " - " + typeMatches[1]
			}
        }

        if line == "END:VEVENT" {
            if buffering {
                fmt.Fprint(w, buffer.String())
                buffering = false
            }
        } else {
            summaryMatches := summaryRegex.FindStringSubmatch(line)
			// Ignore unwanted courses
            if len(summaryMatches) >= 2 && !courseCodesMap[summaryMatches[1]] {
                ignoring = true
            }
        }

		if strings.HasPrefix(line, " ") {
			// Ignore useless Summary lines
			continue
		}
        if !buffering {
            fmt.Fprintln(w, line)
        } else {
            buffer.WriteString(line + "\n")
        }
	}
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":" + os.Getenv("PORT"), nil)
}
