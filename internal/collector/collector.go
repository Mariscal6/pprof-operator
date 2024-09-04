package collector

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/pprof/profile"
)

type ProfileCollector interface {
	GetProfile(url string, duration time.Duration, writer io.Writer) error
	MergeProfiles(profiles []io.Reader, writer io.Writer) error
}

var _ ProfileCollector = GoPprofCollector{}

type GoPprofCollector struct{}

// CollectPProf collects pprof data from the given URL for the given duration.
func (c GoPprofCollector) GetProfile(url string, duration time.Duration, writer io.Writer) error {
	res, err := http.Get(fmt.Sprintf("%s?seconds=%f", url, duration.Seconds()))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		msg, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}
		return fmt.Errorf("unexpected status code: %d body %s", res.StatusCode, string(msg))
	}
	defer res.Body.Close()
	tee := io.TeeReader(res.Body, writer)
	_, err = profile.Parse(tee)
	if err != nil {
		return fmt.Errorf("error parsing profile: %v", err)
	}
	return nil
}

func (c GoPprofCollector) MergeProfiles(profiles []io.Reader, writer io.Writer) error {
	if len(profiles) == 0 {
		return fmt.Errorf("no profiles to merge")
	}
	if len(profiles) == 1 {
		_, err := io.Copy(writer, profiles[0])
		return err
	}
	srcs := make([]*profile.Profile, len(profiles))
	for i, p := range profiles {
		src, err := profile.Parse(p)
		if err != nil {
			return fmt.Errorf("error parsing profile: %v", err)
		}
		srcs[i] = src
	}

	merged, err := profile.Merge(srcs)
	if err != nil {
		return fmt.Errorf("error merging profiles: %v", err)
	}

	if err := merged.Write(writer); err != nil {
		return fmt.Errorf("error writing merged profile: %v", err)
	}

	return nil
}
