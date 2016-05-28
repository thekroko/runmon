package main

import (
    "net/http"
    "html/template"
    "io"
    "encoding/csv"
    "strings"
    "fmt"
    "log"
    "os"
    "time"
    "strconv"
)

const (
    port = ":9090"
    indexTemplateString = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>runMon</title>
        <style>
            body { background: #222; color: #CFCFCF; font-family: monospace; padding: 4em; }
            tr:first-child td { font-weight: bold; border-bottom: 1px solid #666; }
            td { min-width: 12em; }
            table { margin-bottom: 1.5em; }
            progress { width: 80%; height: 32px; }
            figure { font-size: 8em; padding: 0; margin: 0; }
            a { color: inherit; }
            h2 { clear: both; margin-top: 2em; }
            body > div { width: 500px; }
            div > div { 
              font-family: sans-serif; color: #FFF; font-size: 0.75em; text-shadow: #000 0 0 2px, #000 0 0 2px; width: 64px; height: 40px; 
              flex: 0 0 auto; padding: 1px; border: 1px solid #FFF; text-align: center; margin: 1px;
            }
            div[data-day=Mon] {
              border-left: 2px solid #F00;
            }
        </style>
	</head>
	<body>
	<figure>🏃</figure>
	<h1>run, run, run!</h1>

        <h2>look at that progress</h2>
        <progress max="365" value="{{.TotalDistanceKm}}"></progress>
	<p>So far, you ran <b>{{.TotalDistanceKm | km}} km</b> of <b>365 km</b></p>

        <h2>schedule</h2>
        <div style="display: flex; flex-flow: row wrap">
 	{{range .Schedule}}
          <div data-day="{{.When.Format "Mon"}}" style="background: {{if eq .DistanceKm 0.0}}#444{{else if .Done}}#0C0{{else}}{{if .InPast}}#C00{{else}}#00A{{end}}{{end}}; {{if .Today}}box-shadow: #FF0 0px 0px 2px 2px{{end}}">
             <b>{{.When.Format "Mon"}}</b><br>
             {{if gt .DistanceKm 0.0}}
               {{.DistanceKm | km}}<br>
               km
             {{end}}
          </div>
        {{else}}
        no schedule
        {{end}}
        </div>

        <h2>all runs</h2>
        <table>
            <tr>
                <td>date</td>
                <td>distance</td>
                <td>duration</td>
            </tr>
	    {{range .Runs}}
                <tr>
                    <td>{{.Date.Format "02.01.06"}}</a></td>
                    <td>{{.DistanceKm | km}} km</td>
                    <td>{{.Duration}}</td>
                </tr>
            {{else}}
                <tr><td colspan=3><strong>no runs</strong></tr>
            {{end}}
        </table>
        <i>keep on running!</i>
	</body>
</html>`
)

var (
    logger = log.New(os.Stdout, "", log.Lmicroseconds)
    funcMap = template.FuncMap{
      "km": func(num float64) string { return fmt.Sprintf("%.1f", num) },
    }
    indexTemplate = template.Must(template.New("index").Funcs(funcMap).Parse(indexTemplateString))
)

func reverse(a []*Run) []*Run {
  for i := len(a)/2-1; i >= 0; i-- {
    opp := len(a)-1-i
    a[i], a[opp] = a[opp], a[i]
  }
  return a
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    runs := getAllRuns()
    var totalDist float64 = 0.0
    for _, r := range runs {
      totalDist = totalDist + r.DistanceKm
    }
    schedule := getSchedule(runs, totalDist)

    templateData := &struct{
      TotalDistanceKm float64
      Runs []*Run
      Schedule []*Schedule
    } { totalDist, reverse(runs), schedule };
    err := indexTemplate.Execute(w, templateData)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
    logRequest := func(str string, args... interface{}) { logger.Printf("Request(%s) from %s: %s", r.URL.Path, r.RemoteAddr, fmt.Sprintf(str, args...)) }
    if r.URL.Path == "/" {
        // Serve Index
        logRequest("index")
        indexHandler(w, r)
        return
    }

    logRequest("not found")
    http.Error(w, "not found", http.StatusNotFound)
}

// ---------------------- Model -------------------

type Run struct {
	Date time.Time
	DistanceKm float64
	Duration time.Duration
}

type Schedule struct {
	Done bool
	InPast bool
        Today bool
        When time.Time
	DistanceKm float64
}

func getAllRuns() []*Run {
   fs, _ := os.Open("tracks.csv")
   defer fs.Close()
   
r := csv.NewReader(fs)

   res := []*Run{}
   for {
     // Read
     records, err := r.Read()
     if err == io.EOF {
       return res
     }
     dateStr := records[1]
     distStr := records[2]
     durStr  := records[3]

     // Parse
     date, _ := time.Parse("1/2/2006", dateStr)
     if !strings.Contains(distStr, "mile") {
	panic("wrong unit: "+distStr)
     }
     dist, _ := strconv.ParseFloat(strings.Replace(distStr," mile","", 1), 64)
     dist = dist * 1.60934
     duration, _ := time.ParseDuration(strings.Replace(durStr, ":", "m", 1) + "s")

     // Output
     newRun := &Run{
       Date: date,
       DistanceKm: dist,
       Duration: duration,
     }
     res = append(res, newRun)
   }
}

func lerp(start, end, progressVal, progressMax float64) float64 {
  progress := progressVal / progressMax
  if progress > 1.0 {
    progress = 1.0
  }
  return start + (end - start) * progress
}

func getSchedule(runs []*Run, totalKm float64) []*Schedule {
  schedule := []*Schedule{}
  startDay := time.Date(2016, time.May, 30, 0, 0, 0, 0, time.UTC)
  endDay := time.Now().Add(time.Hour * 24 * (31 * 4 - 4))
  day := startDay.Add(time.Hour * 24 * -7)
  for {
    if day.After(endDay) {
      break
    }
  
    // figure out schedule
    newSchedule := &Schedule{}
    newSchedule.When = day
    weeks := float64(day.Sub(startDay) / time.Hour / 24.0 / 7.0)
    switch day.Weekday() {
      case time.Tuesday, time.Thursday:
        newSchedule.DistanceKm = lerp(4.0, 7.0, weeks, 2)
      case time.Saturday:
        newSchedule.DistanceKm = lerp(6.0, 10.0, weeks, 5)
      case time.Sunday:
        newSchedule.DistanceKm = lerp(7.0, 25.0, weeks, 16)
    }
    newSchedule.Today = (day.YearDay() == time.Now().YearDay())
    newSchedule.InPast = time.Now().After(day)
    if startDay.After(day) {
      newSchedule.DistanceKm = 0
    }
    schedule = append(schedule, newSchedule)

    // next day
    day = day.Add(time.Hour * 24)
  }

  // Calculate what is done
  var kmSoFar float64 = 0.0
  for _, s := range schedule {
    kmSoFar = kmSoFar + s.DistanceKm
    s.Done = totalKm > kmSoFar * 0.9
  }
  return schedule
}

// ------------------------------------------------

func main() {
    // Spin up webserver
    logger.Printf("Listening on %s\n", port)
    http.HandleFunc("/", requestHandler)
    http.ListenAndServe(port, nil)
    logger.Printf("All done\n")
}
