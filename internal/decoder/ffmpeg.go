package decoder

import "fmt"

type FFmpegConfig struct {
	RTSPURL     string
	CaptureFPS  int
	DecodeScale string
}

func BuildFFmpegArgs(cfg FFmpegConfig) []string {
	fps := cfg.CaptureFPS
	if fps <= 0 {
		fps = 25
	}
	scale := cfg.DecodeScale
	if scale == "" {
		scale = "1920x1080"
	}

	filter := fmt.Sprintf("fps=%d,scale=%s", fps, scale)

	return []string{
		"ffmpeg",
		"-rtsp_transport", "tcp",
		"-rtsp_flags", "prefer_tcp",
		"-stimeout", "5000000",
		"-i", cfg.RTSPURL,
		"-f", "rawvideo",
		"-pix_fmt", "bgr24",
		"-vf", filter,
		"pipe:1",
	}
}
