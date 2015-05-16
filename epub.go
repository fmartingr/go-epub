package epub

import "github.com/satori/go.uuid"

const (
	urnUuid = "urn:uuid:"
)

type epub struct {
	author string
	lang   string
	pkgdoc *pkgdoc
	title  string
	toc    *toc
	uuid   string
}

func NewEpub(title string) (*epub, error) {
	var err error

	e := &epub{}
	e.pkgdoc = newPkgdoc()
	e.toc, err = newToc()
	if err != nil {
		return e, err
	}
	// Set minimal required attributes
	e.SetLang("en")
	e.SetTitle(title)
	e.SetUUID(urnUuid + uuid.NewV4().String())

	return e, nil
}

func (e *epub) Lang() string {
	return e.lang
}

func (e *epub) SetAuthor(author string) {
	e.pkgdoc.setAuthor(author)
}

func (e *epub) SetLang(lang string) {
	e.lang = lang
	e.pkgdoc.setLang(lang)
}

func (e *epub) SetTitle(title string) {
	e.title = title
	e.pkgdoc.setTitle(title)
	e.toc.setTitle(title)
}

func (e *epub) SetUUID(uuid string) {
	e.uuid = uuid
	e.pkgdoc.setUUID(uuid)
	e.toc.setUUID(uuid)
}

func (e *epub) Title() string {
	return e.title
}

func (e *epub) Uuid() string {
	return e.uuid
}
