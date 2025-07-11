package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/olegshulyakov/go-briefly-bot/briefly/transcript/utils"
)

var (
	ytDlpAdditionalOptions []string
)

func init() {
	ytDlpAdditionalOptions = strings.Fields(os.Getenv("YT_DLP_ADDITIONAL_OPTIONS"))
}

// VideoInfo represents metadata about a YouTube video.
type VideoInfo struct {
	ID        string `json:"id"`        // The unique identifier of the video.
	Language  string `json:"language"`  // The video language.
	Uploader  string `json:"uploader"`  // The name of the video uploader.
	Title     string `json:"title"`     // The title of the video.
	Thumbnail string `json:"thumbnail"` // The URL of the video's thumbnail.
}

// GetYoutubeVideoInfo retrieves metadata about a YouTube video using its URL.
//
// The function uses the `yt-dlp` command-line tool to extract video information
// in JSON format and then parses it into a VideoInfo struct.
//
// Parameters:
//   - videoURL: The URL of the YouTube video.
//
// Returns:
//   - A VideoInfo struct containing the video's metadata.
//   - An error if the video information cannot be extracted or parsed.
//
// Example:
//
//	videoInfo, err := GetYoutubeVideoInfo("https://www.youtube.com/watch?v=example")
//	if err != nil {
//	    log.Errorf("Failed to get video info: %v", err)
//	}
//	fmt.Printf("Video Title: %s\n", videoInfo.Title)
//
// Notes:
//   - The function relies on the `yt-dlp` tool being installed and accessible in the system's PATH.
func GetYoutubeVideoInfo(videoURL string) (*VideoInfo, error) {
	if videoURL == "" {
		return nil, errors.New("videoURL is empty")
	}
	if !IsValidYouTubeURL(videoURL) {
		return nil, fmt.Errorf("no valid URL found: %s", videoURL)
	}

	var videoInfo *VideoInfo

	slog.Debug("VideoInfo download", "url", videoURL)

	args := []string{
		"--dump-json",
	}

	output, err := execYtDlp(args, videoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract video info: %w", err)
	}

	err = json.Unmarshal(output, &videoInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	slog.Debug("VideoInfo downloaded", "url", videoURL)

	return videoInfo, nil
}

// GetYoutubeTranscript retrieves the transcript of a YouTube video using its URL.
//
// The function uses the `yt-dlp` command-line tool to extract the transcript
// in SRT format, reads the transcript file, and then deletes the file.
//
// Parameters:
//   - videoURL: The URL of the YouTube video.
//
// Returns:
//   - A string containing the transcript of the video.
//   - An error if the transcript cannot be extracted, read, or the file cannot be deleted.
//
// Example:
//
//	transcript, err := GetYoutubeTranscript("https://www.youtube.com/watch?v=example")
//	if err != nil {
//	    log.Errorf("Failed to get transcript: %v", err)
//	}
//	fmt.Println("Transcript:", transcript)
//
// Notes:
//   - The function relies on the `yt-dlp` tool being installed and accessible in the system's PATH.
//   - The transcript is extracted in Russian (`ru` and `ru_auto`) and saved as an SRT file.
//   - The transcript file is deleted after reading to clean up temporary files.
func GetYoutubeTranscript(videoURL string, languageCode string) (string, error) {
	const extension = "srt"

	if videoURL == "" {
		return "", errors.New("videoURL is empty")
	}
	if languageCode != "" {
		languageCode = "en"
	}
	if !IsValidYouTubeURL(videoURL) {
		return "", fmt.Errorf("no valid URL found: %s", videoURL)
	}

	videoID, err := GetYouTubeID(videoURL)
	if err != nil {
		return "", err
	}

	args := []string{
		"--no-progress",
		"--skip-download",
		"--write-subs",
		"--write-auto-subs",
		"--convert-subs", extension,
		"--sub-lang", fmt.Sprintf("%s,%s_auto,-live_chat", languageCode, languageCode),
		"--output", fmt.Sprintf("subtitles_%s.%%(ext)s", videoID),
	}

	slog.Debug("Transcript extract", "url", videoURL)
	output, err := execYtDlp(args, videoURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract transcript: %w\n%s", err, output)
	}

	// Read the transcript file
	transcript, err := utils.ReadAndRemoveFile(fmt.Sprintf("subtitles_%s.%s.%s", videoID, languageCode, extension))
	if err != nil {
		return "", err
	}

	slog.Debug("Transcript extracted", "url", videoURL)

	cleaned, err := utils.CleanSRT(transcript)
	if err != nil {
		return "", fmt.Errorf("failed to clean transcript file: %w", err)
	}

	slog.Debug("Transcript cleaned", "url", videoURL)

	return cleaned, nil
}

func execYtDlp(arguments []string, url string) ([]byte, error) {
	const maxAttempts = 3
	var (
		err    error
		output []byte
		args   = make([]string, 0, len(arguments)+len(ytDlpAdditionalOptions)+1)
	)

	if len(ytDlpAdditionalOptions) > 0 {
		args = append(args, ytDlpAdditionalOptions...)
	}
	args = append(args, arguments...)
	args = append(args, url)

	// Execute with retry
	for attempt := 0; attempt < maxAttempts; attempt++ {
		cmd := exec.Command("yt-dlp", args...)
		output, err = cmd.Output()
		if err == nil {
			break
		}
	}

	if err != nil {
		return output, fmt.Errorf("failed to extract transcript: %w\n%s", err, output)
	}

	return output, nil
}
