package transformer

//import (
//	"fmt"
//	"github.com/notedit/gst"
//)
//
//const (
//	aac2opusPipeline = "appsrc name=appsrc is-live=true ! decodebin ! audioconvert ! audioresample ! opusenc ! appsink max-buffers=1 name=audiosink"
//)
//
//func init() {
//	// init the main loop
//}
//
//// audio
//type GstTransformer struct {
//	pipeline  *gst.Pipeline
//	audiosrc  *gst.Element
//	audiosink *gst.Element
//}
//
//func (g *GstTransformer) Open() (err error) {
//	g.pipeline, err = gst.ParseLaunch(aac2opusPipeline)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	audiosrc := g.pipeline.GetByName("appsrc")
//	audiosink := g.pipeline.GetByName("audiosink")
//	g.audiosrc = audiosrc
//	g.audiosink = audiosink
//	g.pipeline.SetState(gst.StatePlaying)
//	g.pipeline.SetLatency(0)
//	return
//}
//
//func (g *GstTransformer) Push(data []byte) error {
//	err := g.audiosrc.PushBuffer(data)
//	return err
//}
//
//func (g *GstTransformer) PullSample() ([]byte, error) {
//	sample,err := g.audiosink.PullSample()
//	if err != nil {
//		return nil, err
//	}
//	return sample.Data,nil
//}
//
//func (g *GstTransformer) Stop() {
//	if g.pipeline != nil {
//		g.pipeline.SetState(gst.StateNull)
//		g.pipeline = nil
//	}
//	g.audiosrc = nil
//	g.audiosink = nil
//}
//





