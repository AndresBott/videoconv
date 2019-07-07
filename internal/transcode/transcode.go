package transcode

import (
	"git.andresbott.com/Utilities/go-video-process/internal/config"
	"github.com/AndresBott/f/fm"
	"github.com/AndresBott/f/fm/dir"
	"github.com/AndresBott/f/fm/file"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Transcoder struct {
	cf            config.Conf
	video         *fmfile.File
	videoSettings []config.VideoSetting
}

func NewTranscoder() *Transcoder {
	t := Transcoder{
		cf: config.NewConfig(),
	}
	return &t
}

func (t *Transcoder) Interval() int {
	return t.cf.PollInterval
}

func (t *Transcoder) Run() {

	t.discoverFile()

	if t.video == nil {
		return
	}

	err := t.prepare()
	if err != nil {
		log.Error(err)
		t.handleTranscodeError()
		return
	}

	t.transcodeVideo()
}

// scan the input directory and get the first found video file
func (t *Transcoder) discoverFile() {

	t.video = nil
	exts := t.cf.Ext()
	log.Info("searching video files in: " + t.cf.InputFolder + " with extension: " + strings.Join(exts, ","))

	inDir, err := fmdir.NewDir(t.cf.InputFolder)
	if err != nil {
		log.Warn("input dir: " + err.Error())
		return
	}

	opts := fmdir.DirScanOpts{
		FilterExtensions: exts,
		MaxSubLevels:10,
		MaxResults:1,

	}

	err2 := inDir.Scan(opts)
	if err2 != nil {
		log.Warn(err2)
	}

	if len(inDir.Files()) > 0 {
		t.video = &inDir.Files()[0]
		log.Info("found video: \"" + t.video.Name() + "\"")
	}

}

// prepare some struc data before running execution
func (t *Transcoder) prepare() error {
	var err error

	t.videoSettings, err = t.cf.VideoSettingsByExt(t.video.Ext)

	if err != nil {
		return err
	}

	return nil
}

func (t *Transcoder) transcodeVideo() {

	log.Info("transcoding video file:" + t.video.FullPath())
	fileManager, _ := fm.NewFm("")

	timeStart := time.Now()

	// loop over video settings and transcode them
	for _, setting := range t.videoSettings {

		videoTmpOut := t.cf.TmpDir + "/" + t.video.Basename() + "_" + setting.Name() + "." + setting.OutputExtension()
		cmd := setting.Cmd()

		// make sure the tmp file does not exist
		err := fileManager.DeleteFile(videoTmpOut)
		if err != nil {
			log.Error(err)
		}

		err = t.runffmpeg(cmd, videoTmpOut)
		if err != nil {
			log.Error("error in ffmpeg, rolling back", err)
			t.handleTranscodeError()
			return
		}
	}

	// move finalized files
	relativePath,err := filepath.Rel(t.cf.InputFolder,t.video.FullPath())
	if err != nil {
		log.Error(err)
		t.handleTranscodeError()
	}
	relativePath = filepath.Dir(relativePath)
	outPutPath := filepath.Clean(t.cf.OutputFolder+"/"+relativePath)

	err = os.MkdirAll(outPutPath,0750)
	if err != nil {
		log.Error(err)
		t.handleTranscodeError()
		return
	}

	for _, setting := range t.videoSettings {

		videoTmpOut := filepath.Clean(t.cf.TmpDir + "/" + t.video.Basename() + "_" + setting.Name() + "." + setting.OutputExtension())
		videoOut := filepath.Clean(outPutPath + "/" + t.video.Basename() + "_" + setting.Name() + "." + setting.OutputExtension())

		// make sure the destination file does not exist
		err := fileManager.MoveFile(videoTmpOut, videoOut, false)
		if err != nil {
			log.Error(err)
			t.handleTranscodeError()
			return
		}
	}
	// move the original


	err = fileManager.MoveFile(t.video.FullPath(), filepath.Clean(outPutPath+"/"+t.video.Name()), false)
	if err != nil {
		log.Error(err)
		t.handleTranscodeError()
	}
	timeEnd := time.Now()
	timeDiff := timeEnd.Sub(timeStart)
	log.Info("finished processing: \"" + t.video.Name() + "\" in " + timeDiff.String())

}

func (t *Transcoder) handleTranscodeError() {
	log.Warn("handling transcode Error")

	fileManager, _ := fm.NewFm("")
	for _, setting := range t.videoSettings {

		videoTmpOut := t.cf.TmpDir + "/" + t.video.Basename() + "_" + setting.Name() + "." + setting.OutputExtension()

		// make sure the tmp file does not exist
		log.Warn("deleting possible tmp file: \"" + videoTmpOut + "\"")
		err := fileManager.DeleteFile(videoTmpOut)
		if err != nil {
			log.Fatal(err)
		}
	}
	// move the original to ignore

	log.Warn("moving file: " + t.video.FullPath() + " to ignore dir")
	err := fileManager.MoveFile(t.video.FullPath(), t.cf.IgnoreDir+"/"+t.video.Name(), false)
	if err != nil {
		log.Error(err)
	}

}

func (t *Transcoder) runffmpeg(cmd string, videoTmpOut string) error {
	log.Info("rendering video setting: " + t.video.Name() + " -> " + videoTmpOut)
	binary := "/usr/bin/ffmpeg"

	args := []string{}

	args = append(args, "-i")
	args = append(args, t.video.FullPath())

	args = append(args, "-threads")
	args = append(args, strconv.Itoa(t.cf.ProcessorThreads))

	se := strings.Split(cmd, " ")
	for _, s := range se {
		args = append(args, strings.Trim(s, "'"))
	}

	args = append(args, videoTmpOut)

	e := exec.Command(binary, args...)
	//e.Stdout = os.Stdout
	//e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return err
	}
	return nil
}