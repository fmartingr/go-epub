package epub

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testAuthorTemplate    = `<dc:creator id="creator">%s</dc:creator>`
	testContainerContents = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="EPUB/package.opf" media-type="application/oebps-package+xml" />
  </rootfiles>
</container>`
	testDirPerm               = 0775
	testEpubAuthor            = "Hingle McCringleberry"
	testEpubcheckJarfile      = "epubcheck.jar"
	testEpubcheckPrefix       = "epubcheck"
	testEpubFilename          = "My EPUB.epub"
	testEpubLang              = "fr"
	testEpubTitle             = "My title"
	testEpubUUID              = "51b7c9ea-b2a2-49c6-9d8c-522790786d15"
	testImageFromFileFilename = "testfromfile.png"
	testImageFromFileSource   = "testdata/gophercolor16x16.png"
	testImageFromURLSource    = "https://golang.org/doc/gopher/gophercolor16x16.png"
	testLangTemplate          = `<dc:language>%s</dc:language>`
	testMimetypeContents      = "application/epub+zip"
	testPkgContentTemplate    = `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="pub-id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="pub-id">urn:uuid:21ed94b4-f2ab-44c8-b99d-4f7792587ad6</dc:identifier>
    <dc:title>%s</dc:title>
    <dc:language>en</dc:language>
    <meta property="dcterms:modified">2016-04-28T19:09:26Z</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"></item>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"></item>
  </manifest>
  <spine toc="ncx"></spine>
</package>`
	testSectionBody = `    <h1>Section 1</h1>
	<p>This is a paragraph.</p>`
	testSectionContentTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>%s</title>
  </head>
  <body>
    %s
  </body>
</html>`
	testSectionFilename = "section0001.xhtml"
	testSectionTitle    = "Section 1"
	testTempDirPrefix   = "go-epub"
	testTitleTemplate   = `<dc:title>%s</dc:title>`
	testUUIDTemplate    = `<dc:identifier id="pub-id">urn:uuid:%s</dc:identifier>`
)

func TestEpubWrite(t *testing.T) {
	e := NewEpub(testEpubTitle)

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	// Check the contents of the mimetype file
	contents, err := ioutil.ReadFile(filepath.Join(tempDir, mimetypeFilename))
	if err != nil {
		t.Errorf("Unexpected error reading mimetype file: %s", err)
	}
	if trimAllSpace(string(contents)) != trimAllSpace(testMimetypeContents) {
		t.Errorf(
			"Mimetype file contents don't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			contents,
			testMimetypeContents)
	}

	// Check the contents of the container file
	contents, err = ioutil.ReadFile(filepath.Join(tempDir, metaInfFolderName, containerFilename))
	if err != nil {
		t.Errorf("Unexpected error reading container file: %s", err)
	}
	if trimAllSpace(string(contents)) != trimAllSpace(testContainerContents) {
		t.Errorf(
			"Container file contents don't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			contents,
			testContainerContents)
	}

	// Check the contents of the package file
	contents, err = ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	testPkgContents := fmt.Sprintf(testPkgContentTemplate, testEpubTitle)
	if trimAllSpace(string(contents)) != trimAllSpace(testPkgContents) {
		t.Errorf(
			"Package file contents don't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			contents,
			testPkgContents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestAddImage(t *testing.T) {
	e := NewEpub(testEpubTitle)
	_, err := e.AddImage(testImageFromFileSource, testImageFromFileFilename)
	if err != nil {
		t.Errorf("Error adding image: %s", err)
	}

	testImageFromURLPath, err := e.AddImage(testImageFromURLSource, "")
	if err != nil {
		t.Errorf("Error adding image: %s", err)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, imageFolderName, testImageFromFileFilename))
	if err != nil {
		t.Errorf("Unexpected error reading image file from EPUB: %s", err)
	}

	testImageContents, err := ioutil.ReadFile(testImageFromFileSource)
	if err != nil {
		t.Errorf("Unexpected error reading testdata image file: %s", err)
	}
	if bytes.Compare(contents, testImageContents) != 0 {
		t.Errorf("Image file contents don't match")
	}

	// The image path is relative to the XHTML folder
	contents, err = ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, xhtmlFolderName, testImageFromURLPath))
	if err != nil {
		t.Errorf("Unexpected error reading image file from EPUB: %s", err)
	}

	resp, err := http.Get(testImageFromURLSource)
	if err != nil {
		t.Errorf("Unexpected error response from test image URL: %s", err)
	}
	testImageContents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Unexpected error reading test image file from URL: %s", err)
	}
	if bytes.Compare(contents, testImageContents) != 0 {
		t.Errorf("Image file contents don't match")
	}

	cleanup(testEpubFilename, tempDir)
}

func TestAddSection(t *testing.T) {
	e := NewEpub(testEpubTitle)
	_, err := e.AddSection(testSectionTitle, testSectionBody, testSectionFilename)
	if err != nil {
		t.Errorf("Error adding section: %s", err)
	}

	testSectionPath, err := e.AddSection(testSectionTitle, testSectionBody, "")
	if err != nil {
		t.Errorf("Error adding section: %s", err)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, xhtmlFolderName, testSectionFilename))
	if err != nil {
		t.Errorf("Unexpected error reading section file: %s", err)
	}

	testSectionContents := fmt.Sprintf(testSectionContentTemplate, testSectionTitle, testSectionBody)
	if trimAllSpace(string(contents)) != trimAllSpace(testSectionContents) {
		t.Errorf(
			"Section file contents don't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			contents,
			testSectionContents)
	}

	contents, err = ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, xhtmlFolderName, testSectionPath))
	if err != nil {
		t.Errorf("Unexpected error reading section file: %s", err)
	}

	if trimAllSpace(string(contents)) != trimAllSpace(testSectionContents) {
		t.Errorf(
			"Section file contents don't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			contents,
			testSectionContents)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubAuthor(t *testing.T) {
	e := NewEpub(testEpubTitle)
	e.SetAuthor(testEpubAuthor)

	if e.Author() != testEpubAuthor {
		t.Errorf(
			"Author doesn't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			e.Author(),
			testEpubAuthor)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	testAuthorElement := fmt.Sprintf(testAuthorTemplate, testEpubAuthor)
	if !strings.Contains(string(contents), testAuthorElement) {
		t.Errorf(
			"Author doesn't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			contents,
			testAuthorElement)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubLang(t *testing.T) {
	e := NewEpub(testEpubTitle)
	e.SetLang(testEpubLang)

	if e.Lang() != testEpubLang {
		t.Errorf(
			"Language doesn't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			e.Lang(),
			testEpubLang)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	testLangElement := fmt.Sprintf(testLangTemplate, testEpubLang)
	if !strings.Contains(string(contents), testLangElement) {
		t.Errorf(
			"Language doesn't match\n"+
				"Got: %s"+
				"Expected: %s",
			contents,
			testLangElement)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubTitle(t *testing.T) {
	// First, test the title we provide when creating the epub
	e := NewEpub(testEpubTitle)
	if e.Title() != testEpubTitle {
		t.Errorf(
			"Title doesn't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			e.Title(),
			testEpubTitle)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	testTitleElement := fmt.Sprintf(testTitleTemplate, testEpubTitle)
	if !strings.Contains(string(contents), testTitleElement) {
		t.Errorf(
			"Title doesn't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			contents,
			testTitleElement)
	}

	cleanup(testEpubFilename, tempDir)

	// Now test changing the title
	e.SetTitle(testEpubAuthor)

	if e.Title() != testEpubAuthor {
		t.Errorf(
			"Title doesn't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			e.Title(),
			testEpubAuthor)
	}

	tempDir = writeAndExtractEpub(t, e, testEpubFilename)

	contents, err = ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	testTitleElement = fmt.Sprintf(testTitleTemplate, testEpubAuthor)
	if !strings.Contains(string(contents), testTitleElement) {
		t.Errorf(
			"Title doesn't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			contents,
			testTitleElement)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubUUID(t *testing.T) {
	e := NewEpub(testEpubTitle)
	e.SetUUID(testEpubUUID)

	if e.UUID() != testEpubUUID {
		t.Errorf(
			"UUID doesn't match\n"+
				"Got: %s\n"+
				"Expected: %s",
			e.UUID(),
			testEpubUUID)
	}

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	contents, err := ioutil.ReadFile(filepath.Join(tempDir, contentFolderName, pkgFilename))
	if err != nil {
		t.Errorf("Unexpected error reading package file: %s", err)
	}

	testUUIDElement := fmt.Sprintf(testUUIDTemplate, testEpubUUID)
	if !strings.Contains(string(contents), testUUIDElement) {
		t.Errorf(
			"UUID doesn't match\n"+
				"Got: %s"+
				"Expected: %s",
			contents,
			testUUIDElement)
	}

	cleanup(testEpubFilename, tempDir)
}

func TestEpubValidity(t *testing.T) {
	e := NewEpub(testEpubTitle)
	e.AddImage(testImageFromFileSource, testImageFromFileFilename)
	e.AddImage(testImageFromURLSource, "")
	e.AddSection(testSectionTitle, testSectionBody, testSectionFilename)
	e.AddSection(testSectionTitle, testSectionBody, "")
	e.SetAuthor(testEpubAuthor)
	e.SetLang(testEpubLang)
	e.SetTitle(testEpubAuthor)
	e.SetUUID(testEpubUUID)

	tempDir := writeAndExtractEpub(t, e, testEpubFilename)

	output, err := validateEpub(t, testEpubFilename)
	if err != nil {
		t.Errorf("EPUB validation failed")
	}

	// Always print the output so we can see warnings as well
	fmt.Println(string(output))

	cleanup(testEpubFilename, tempDir)
}

func cleanup(epubFilename string, tempDir string) {
	os.Remove(epubFilename)
	os.RemoveAll(tempDir)
}

// TrimAllSpace trims all space from each line of the string and removes empty
// lines for easier comparison
func trimAllSpace(s string) string {
	trimmedLines := []string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			trimmedLines = append(trimmedLines)
		}
	}

	return strings.Join(trimmedLines, "\n")
}

// UnzipFile unzips a file located at sourceFilePath to the provided destination directory
func unzipFile(sourceFilePath string, destDirPath string) error {
	// First, make sure the destination exists and is a directory
	info, err := os.Stat(destDirPath)
	if err != nil {
		return err
	}
	if !info.Mode().IsDir() {
		return errors.New("destination is not a directory")
	}

	r, err := zip.OpenReader(sourceFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	// Iterate through each file in the archive
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		destFilePath := filepath.Join(destDirPath, f.Name)

		// Create destination subdirectories if necessary
		destBaseDirPath, _ := filepath.Split(destFilePath)
		os.MkdirAll(destBaseDirPath, testDirPerm)

		// Create the destination file
		w, err := os.Create(destFilePath)
		if err != nil {
			return err
		}
		defer func() {
			if err := w.Close(); err != nil {
				panic(err)
			}
		}()

		// Copy the contents of the source file
		_, err = io.Copy(w, rc)
		if err != nil {
			return err
		}
	}

	return nil
}

// This function requires epubcheck to work (https://github.com/IDPF/epubcheck)
//
//     wget https://github.com/IDPF/epubcheck/releases/download/v4.0.1/epubcheck-4.0.1.zip
//     unzip epubcheck-4.0.1.zip
func validateEpub(t *testing.T, epubFilename string) ([]byte, error) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Error("Error getting working directory")
	}

	items, err := ioutil.ReadDir(cwd)
	if err != nil {
		t.Error("Error getting contents of working directory")
	}

	pathToEpubcheck := ""
	for _, i := range items {
		if i.Name() == testEpubcheckJarfile {
			pathToEpubcheck = i.Name()
			break

		} else if strings.HasPrefix(i.Name(), testEpubcheckPrefix) {
			if i.Mode().IsDir() {
				pathToEpubcheck = filepath.Join(i.Name(), testEpubcheckJarfile)
				if _, err := os.Stat(pathToEpubcheck); err == nil {
					break
				} else {
					pathToEpubcheck = ""
				}
			}
		}
	}

	if pathToEpubcheck == "" {
		fmt.Println("Epubcheck tool not installed, skipping EPUB validation.")
		return []byte{}, nil
	}

	cmd := exec.Command("java", "-jar", pathToEpubcheck, epubFilename)
	return cmd.CombinedOutput()
}

func writeAndExtractEpub(t *testing.T, e *Epub, epubFilename string) string {
	tempDir, err := ioutil.TempDir("", tempDirPrefix)
	if err != nil {
		t.Errorf("Unexpected error creating temp dir: %s", err)
	}

	err = e.Write(epubFilename)
	if err != nil {
		t.Errorf("Unexpected error writing EPUB: %s", err)
	}

	err = unzipFile(epubFilename, tempDir)
	if err != nil {
		t.Errorf("Unexpected error extracting EPUB: %s", err)
	}

	return tempDir
}