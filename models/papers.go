package models

type BingPapers struct {
	Images []Image `json:"images"`
}

type Image struct {
	Startdate     string        `json:"startdate"`
	Fullstartdate string        `json:"fullstartdate"`
	Enddate       string        `json:"enddate"`
	URL           string        `json:"url"`
	Urlbase       string        `json:"urlbase"`
	Copyright     string        `json:"copyright"`
	Copyrightlink string        `json:"copyrightlink"`
	Title         string        `json:"title"`
}
