package gallerysync

import (
	"github.com/clouderhem/micloud/micloud/gallery/album"
	"github.com/clouderhem/micloud/micloud/gallery/gallery"
)

type Timeline struct {
	StartDate int
	EndDate   int
	Count     int
}

type AlbumsWrapper struct {
	Album     album.Album
	Galleries []gallery.Gallery
}
