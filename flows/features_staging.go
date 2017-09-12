package flows

/*

Features in here are subject to change. Use them with caution.

*/

////////////////////////////////////////////////////////////////////////////////

type _activeForSeconds struct {
	BaseFeature
	count Unsigned64
    last_time int64  // FIXME maybe this should be dateTimeSeconds?
}

func (f *_activeForSeconds) Start(context EventContext) {
    f.last_time = 0
    f.count = 0
}

func (f *_activeForSeconds) Event(new interface{}, context EventContext, src interface{}) {
    var time int64
    if f.last_time == 0 {
        f.last_time = time
        f.count++
    } else {
        if time - f.last_time > 1000000000 {  // if time difference to f.last_time is more than one second
            f.last_time = time
            f.count++
        }
    }
}

func (f *_activeForSeconds) Stop(reason FlowEndReason, context EventContext) {
	f.SetValue(f.count, context, f)
}

func init() {
	RegisterFeature("_activeForSeconds", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &_activeForSeconds{} }, []FeatureType{RawPacket}},
	})
}
