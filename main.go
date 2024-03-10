package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gocolly/colly"
)

type Note struct {
	Id      string `json:"id"`
	Url     string `json:"url"`
	Grammar string `json:"grammar"`
	Reading string `json:"reading"`
	Meaning string `json:"meaning"`
}

func main() {
	level := flag.String("level", "N2", "JLPT level (N1, N2, N3, N4, N5)")
	flag.Parse()

	// Mapping of JLPT level to the number of pages to scrape present in jlpt-sensei
	levelPages := map[string]int{
		"N1": 7,
		"N2": 5,
		"N3": 5,
		"N4": 4,
		"N5": 3,
	}

	pages, ok := levelPages[*level]
	if !ok {
		log.Fatalf("Invalid JLPT level: %s", *level)
	}

	allNotes := make([]Note, 0)

	collector := colly.NewCollector(
		colly.AllowedDomains("jlptsensei.com", "www.jlptsensei.com"),
	)

	// Register the event handlers outside the loop
	collector.OnHTML("tbody tr.jl-row", func(element *colly.HTMLElement) {
		id := element.ChildText("td.jl-td-num")
		grammar := element.ChildText("td.jl-td-gj a.jl-link")
		reading := element.ChildText("td.jl-td-gr a.jl-link")
		url := element.ChildAttr("a.jl-link", "href")
		meaning := element.ChildText("td.jl-td-gm")

		note := Note{
			Id:      id,
			Grammar: grammar,
			Url:     url,
			Reading: reading,
			Meaning: meaning,
		}

		allNotes = append(allNotes, note)
	})

	collector.OnRequest(func(request *colly.Request) {
		fmt.Println("Visiting", request.URL.String())
	})

	// Loop through all pages
	for i := 1; i <= pages; i++ {
		url := fmt.Sprintf("https://jlptsensei.com/jlpt-%s-grammar-list/page/%d/", *level, i)
		collector.Visit(url)
	}

	writeJSON(allNotes, *level)
}

func writeJSON(data []Note, level string) {
	filename := fmt.Sprintf("jlptnotes_%s.json", level)
	file, err := os.Create(filename)
	if err != nil {
		log.Println("Unable to create the JSON file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	if err := encoder.Encode(data); err != nil {
		log.Println("Unable to encode JSON:", err)
		return
	}
	fmt.Println("Data written to", filename)
}
