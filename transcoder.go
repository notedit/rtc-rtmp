package rtcrtmp

const (
	aac2opus_pipeline = "appsrc do-timestamp=true is-live=true  name=appsrc ! decodebin ! audioconvert ! audioresample ! opusenc ! opusdec ! autoaudiosink "
	opus2aac_pipeline = "appsrc do-timestamp=true is-live=true  name=appsrc ! decodebin ! audioconvert ! audioresample ! opusenc "
)

type AudioFrame struct {
	Channel     int
	SampleRate  int
	SampleCount int
	Data        []byte
}

type Transcorder interface {
}
