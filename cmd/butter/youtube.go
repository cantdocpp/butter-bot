package main

func getStreamUrl(url string) (string, error) {
	video, err := yt.GetVideo(url)
	if err != nil {
		return "", err
	}

	formats := video.Formats.WithAudioChannels()
	streamUrl, err := yt.GetStreamURL(video, &formats[0])
	if err != nil {
		return "", err
	}

	return streamUrl, nil
}
