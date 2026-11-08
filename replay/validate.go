package replay

import "fmt"

// ValidatePlayback checks values that index game data before an untrusted
// user replay reaches a scene. Decode remains a lossless format decoder;
// original overlarge frame-count headers are deliberately accepted here.
func (r *Replay) ValidatePlayback() error {
	if r == nil || len(r.Chapters) == 0 {
		return fmt.Errorf("replay: no playable chapter")
	}
	if r.Global.Character > 1 || r.Global.ShotType > 2 || r.Global.Difficulty > 4 {
		return fmt.Errorf("replay: invalid character, shot type or difficulty")
	}
	if r.Global.Stage < 1 || r.Global.Stage > 8 {
		return fmt.Errorf("replay: invalid final stage %d", r.Global.Stage)
	}
	for i, ch := range r.Chapters {
		if ch.Index < 1 || ch.Index > 7 || len(ch.Records) == 0 {
			return fmt.Errorf("replay: chapter %d is not playable", i)
		}
		if ch.Power < 0 || ch.Power > 100 {
			return fmt.Errorf("replay: chapter %d has invalid power %d", i, ch.Power)
		}
	}
	return nil
}
