package musicserver

import (
	"net/http"
	"strings"
	"os"
	"io"

	"./playlist"
)

func homeHandler(w http.ResponseWriter, req *http.Request) {
	// Check if alias set for this ip
	ip := getIPFromRequest(req)
	if _, aliasSet := al.Alias(ip); !aliasSet {
		http.Redirect(w, req, url("/alias"), http.StatusSeeOther)
		return
	}
	tl.Render(w, "home", newPlaylistInfo(ip))
}

func aliasHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		tl.Render(w, "alias", nil)
		return
	}

	newAlias := strings.TrimSpace(req.PostFormValue("alias_value"))
	if len(newAlias) < 1 {
		http.Redirect(w, req, url("/alias"), http.StatusSeeOther)
		return
	}

	ip := getIPFromRequest(req)
	// Set alias in the manager
	al.SetAlias(ip, newAlias)
	// Update listed aliases in the playlist in new goroutine
	go pl.UpdateAlias(ip, newAlias)

	http.Redirect(w, req, url("/"), http.StatusSeeOther)
}

func queueVideoHandler(w http.ResponseWriter, req *http.Request) {
	ip := getIPFromRequest(req)
	if req.Method != http.MethodPost {
		http.Redirect(w, req, url("/"), http.StatusSeeOther)
		return
	}
	// Check if alias set for this ip
	alias, aliasSet := al.Alias(ip)
	if !aliasSet {
		http.Redirect(w, req, url("/alias"), http.StatusSeeOther)
		return
	}

	if !pl.Available(ip) {
		tl.Render(w, "not_added", "You have the maxium amount of videos queued.")
		return
	}

	newLink := req.PostFormValue("video_link")
	subs := false
	if req.PostFormValue("download_subs") == "on" {
		subs = true
	}

	newVideo := playlist.NewVideo(ip, alias, subs)
	pl.AddVideo(newVideo)
	go downloadVideo(newLink, newVideo.UUID)

	tl.Render(w, "added", nil)
}

func uploadVideoHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Redirect(w, req, url("/"), http.StatusSeeOther)
		return
	}

	// Open video file from post request before redirects can occur or the 
	// connection may be reset.
	file, header, err := req.FormFile("video_file")
	if file == nil {
		tl.Render(w, "not_added", "No file uploaded")
		return
	}
	defer file.Close()

	if err != nil {
		tl.Render(w, "not_added", "Cannot parse uploaded file")
		return
	}

	ip := getIPFromRequest(req)
	// Check if alias set for this ip
	alias, aliasSet := al.Alias(ip)
	if !aliasSet {
		http.Redirect(w, req, url("/alias"), http.StatusSeeOther)
		return
	}

	if !pl.Available(ip) {
		tl.Render(w, "not_added", "You have the maximum amount of videos queued")
		return
	}

	newVid := playlist.NewVideo(ip, alias, false)
	// Gen file path with filename as uuid and get file extension from header
	newPath := vidFolder + "/" + newVid.UUID  + fileExt(header.Filename)

	// Create new file
	newFile, err := os.Create(newPath)
	defer newFile.Close()
	if err != nil {
		tl.Render(w, "not_added", err.Error())
		return
	}

	// Write file
	_, err = io.Copy(newFile, file)
	if err != nil {
		tl.Render(w, "not_added", err.Error())
		return
	}

	// Add information to Video struct
	newVid.Title = stripFileExt(header.Filename)
	newVid.File = newPath
	newVid.Ready = true

	// Add to playlist
	pl.AddVideo(newVid)

	tl.Render(w, "added", nil)
}

func userRemoveHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Redirect(w, req, url("/"), http.StatusSeeOther)
		return
	}

	ip := getIPFromRequest(req)
	uuid := req.PostFormValue("video_id")
	if pl.VideoIP(uuid) == ip {
		pl.RemoveVideo(uuid)
	}
	http.Redirect(w, req, url("/"), http.StatusSeeOther)
}