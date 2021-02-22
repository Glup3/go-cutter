package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jtguibas/cinema"
)

// TS Timestamp
type TS struct {
	start time.Duration
	end   time.Duration
}

func main() {
	//args := os.Args[1:]

	timestamps, err := readTimestamps("./timestamps.txt")

	if err != nil {
		panic(err)
	}

	dir, err := ioutil.TempDir(".", "temp")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	c := make(chan string, len(timestamps))
	var names []string

	vpath := "/Users/phuc/Downloads/valorant2.mp4"

	for i, ts := range timestamps {
		name := fmt.Sprintf("%s/test_valorant%d.mp4", dir, i)

		go createClip(ts.start, ts.end, vpath, name, c)
	}

	for i := 0; i < len(timestamps); i++ {
		names = append(names, <-c)
	}

	clip, err := NewClip(names)

	if err != nil {
		panic(err)
	}

	clip.Concatenate("../test_valorant.mp4")
}

func readTimestamps(filename string) ([]TS, error) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var timestamps []TS
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		ts := strings.Split(scanner.Text(), "-")

		start, err := time.ParseDuration(ts[0])

		if err != nil {
			return nil, err
		}

		end, err := time.ParseDuration(ts[1])

		if err != nil {
			return nil, err
		}

		timestamps = append(timestamps, TS{start: start, end: end})
	}

	return timestamps, nil
}

func createClip(start time.Duration, end time.Duration, videoPath string, filename string, c chan string) {
	video, err := cinema.Load(videoPath)
	if err != nil {
		panic(err)
	}

	video.Trim(start, end)
	video.Render(filename)

	c <- filename
}

// THANK YOU https://github.com/cfanatic/cinema

// Clip contains the absolute or relative path to video files that shall be concatenated.
// Call Clip.NewClip() to initialize the video files and run Clip.Concatenate() to produce
// a single video file.
type Clip struct {
	videosPath      []string
	concatListCache string
}

// NewClip gives you a Clip that can be used to concatenate video files.
// Provide a list of absolute or relative paths to these videos by videoPath.
func NewClip(videoPath []string) (*Clip, error) {
	var clip Clip
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, errors.New("cinema.Load: ffprobe was not found in your PATH " +
			"environment variable, make sure to install ffmpeg " +
			"(https://ffmpeg.org/) and add ffmpeg, ffplay and ffprobe to your " +
			"PATH")
	}

	for _, path := range videoPath {
		if _, err := os.Stat(path); err != nil {
			return nil, errors.New("cinema.Load: unable to load file: " + err.Error())
		}
	}

	dir := filepath.Dir(videoPath[0])
	clip = Clip{videosPath: videoPath, concatListCache: filepath.Join(dir, "concat.txt")}
	return &clip, nil
}

// Concatenate produces a single video clip based on Clip.videosPath and save it as output.
// This method won't return anything on stdout / stderr.
// If you need to read ffmpeg's outputs, use RenderWithStreams.
func (c *Clip) Concatenate(output string) error {
	return c.ConcatenateWithStreams(output, nil, nil)
}

// ConcatenateWithStreams produces a single video clip based on Clip.videosPath and save it as output.
// By specifying an output stream and an error stream, you can read ffmpeg's stdout and stderr.
func (c *Clip) ConcatenateWithStreams(output string, os io.Writer, es io.Writer) error {
	c.saveConcatenateList()
	defer c.deleteConcatenateList()
	line := c.CommandLine(output)
	cmd := exec.Command(line[0], line[1:]...)
	cmd.Stderr = es
	cmd.Stdout = os

	err := cmd.Run()
	if err != nil {
		return errors.New("cinema.Video.Concatenate: ffmpeg failed: " + err.Error())
	}
	return nil
}

// CommandLine returns the command line instruction that will be used to concatenate the video files.
func (c *Clip) CommandLine(output string) []string {
	cmdline := []string{
		"ffmpeg",
		"-y",
		"-f", "concat",
		"-i", c.concatListCache,
		"-c", "copy",
	}
	cmdline = append(cmdline, "-fflags", "+genpts", filepath.Join(filepath.Dir(c.videosPath[0]), output))
	return cmdline
}

func (c *Clip) saveConcatenateList() error {
	f, err := os.Create(c.concatListCache)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, video := range c.videosPath {
		fmt.Fprintf(f, "file '%s'\n", filepath.Base(video))
	}
	return nil
}

func (c *Clip) deleteConcatenateList() error {
	if err := os.Remove(c.concatListCache); err != nil {
		return err
	}
	return nil
}
