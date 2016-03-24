package main

import (
	"encoding/xml"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

var flagDestinationDirectory = flag.String("destdir", "egais_data", "destination directory for store XML files")
var flagServerName = flag.String("server_name", "localhost:8080", "full server name for web server")

var addedFilesList []string

// EgaisURL is a struct for store one record
type EgaisURL struct {
	ReplyID string `xml:"replyId,attr,omitempty"`
	Path    string `xml:",chardata"`
}

// EgaisA is a struct for store all records
type EgaisA struct {
	XMLName xml.Name   `xml:"A"`
	Urls    []EgaisURL `xml:"url"`
}

// ConvertFileNameToURL convert file name to full url
func ConvertFileNameToURL(fileName string) string {
	separator := "/"
	protocol := "http://"
	path := "opt/out"

	fileNameParts := strings.Split(fileName, ".")
	fileName = fileNameParts[0]

	fileNameParts = strings.Split(fileName, "_")
	name := fileNameParts[0]
	id := fileNameParts[1]

	urlData := strings.Join([]string{*flagServerName, path, name, id}, separator)
	urlData = protocol + urlData

	return urlData
}

// FileHasAlreadyAdded search fila name in array added files
func FileHasAlreadyAdded(fileName string) bool {
	for _, f := range addedFilesList {
		if f == fileName {
			return true
		}
	}

	return false
}

// DirToXML read directory and add files from it to A record
func DirToXML(xmlData *EgaisA, path string) error {
	rootDir, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
		return err
	}

	files, err := rootDir.Readdir(0)
	if err != nil {
		log.Fatal(err)
		return err
	}

	for _, fileInfo := range files {
		fullPath := strings.Join([]string{path, fileInfo.Name()}, string(os.PathSeparator))

		if fileInfo.IsDir() {
			dirName := fullPath
			DirToXML(xmlData, dirName)
		} else {
			dirStat, err := os.Stat(path)
			if err != nil {
				log.Fatal(err)
				return err
			}

			var record EgaisURL
			// added replyId if directory not a rootDir
			if path != *flagDestinationDirectory {
				record.ReplyID = dirStat.Name()
			}

			fileStat, err := os.Stat(fullPath)
			if err != nil {
				log.Fatal(err)
				return err
			}
			record.Path = ConvertFileNameToURL(fileStat.Name())

			if !FileHasAlreadyAdded(fileStat.Name()) {
				xmlData.Urls = append(xmlData.Urls, record)
				addedFilesList = append(addedFilesList, fileStat.Name())
			}
		}
	}
	rootDir.Close()

	return nil
}

// GetXMLData return all files in EgaisA struct
func GetXMLData() EgaisA {
	var xmlData EgaisA
	DirToXML(&xmlData, *flagDestinationDirectory)

	return xmlData
}

// Index show index page for web server
func Index(c *gin.Context) {
	c.Status(http.StatusOK)
}

// OutPage show out page of virtual EGAIS
func OutPage(c *gin.Context) {
	out := GetXMLData()
	c.XML(http.StatusOK, out)
}

// GetFile get the file from directory by url
func GetFile(c *gin.Context) {
	name := c.Param("name")
	id := c.Param("id")

	fileName := name + "_" + id + ".xml"
	fullFileName := strings.Join([]string{*flagDestinationDirectory, fileName}, string(os.PathSeparator))

	f, err := os.OpenFile(fullFileName, os.O_RDONLY, 0400)
	if err != nil {
		log.Fatal(err)
		return
	}

	fileStat, err := os.Stat(fullFileName)
	if err != nil {
		log.Fatal(err)
		return
	}

	buffer := make([]byte, fileStat.Size())
	_, err = f.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	c.Data(http.StatusOK, "text/xml", buffer)
}

func main() {
	flag.Parse()

	router := gin.Default()
	router.GET("/", Index)
	router.GET("/opt/out", OutPage)
	router.GET("/opt/out/:name/:id", GetFile)
	router.Run(*flagServerName)
	os.Exit(0)
}
