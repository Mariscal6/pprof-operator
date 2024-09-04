package collector

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/pprof/profile"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestGetProfile(t *testing.T) {
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "localhost:5000/gopher-pprof:latest",
			WaitingFor: wait.ForAll(
				wait.ForHTTP("/healthz").WithPort("8080"),
			),
		},
		Started: true,
		Logger:  testcontainers.TestLogger(t),
	})
	defer container.Terminate(ctx)

	collector := GoPprofCollector{}
	if err != nil {
		t.Fatal(err)
	}

	t.Run("check gettting pprof ", func(t *testing.T) {
		endpoint, err := container.Endpoint(ctx, "http")
		if err != nil {
			t.Fatal(err)
		}
		writer := bytes.Buffer{}
		err = collector.GetProfile(fmt.Sprintf("%s/%s", endpoint, "debug/pprof/profile"), 2*time.Second, &writer)
		if err != nil {
			t.Errorf("CollectPProf() error = %v", err)
			return
		}
		if len(writer.Bytes()) == 0 {
			t.Errorf("CollectPProf() = %v, want non empty", writer.Bytes())
		}
		b := make([]byte, 4)
		n, err := writer.Read(b)
		if err != nil {
			t.Errorf("io.ReadAll() error = %v", err)
			return
		}
		if n != 4 || b[0] != 0x1f || b[1] != 0x8b {
			t.Errorf("CollectPProf() response is not valid")
		}
	})

	// t.Run("check multiple go routines at the same time gettting pprof ", func(t *testing.T) {
	// 	endpoint, err := container.Endpoint(ctx, "http")
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	wg := &sync.WaitGroup{}
	// 	for i := 0; i < 10; i++ {
	// 		wg.Add(1)
	// 		go func() {
	// 			reader, err := collector.GetProfile(fmt.Sprintf("%s/%s", endpoint, "/debug/pprof/profile"), 1*time.Second)
	// 			if err != nil {
	// 				t.Errorf("CollectPProf() error = %v", err)
	// 				return
	// 			}
	// 			b, err := io.ReadAll(reader)
	// 			if err != nil {
	// 				t.Errorf("io.ReadAll() error = %v", err)
	// 				return
	// 			}
	// 			if len(b) == 0 {
	// 				t.Errorf("CollectPProf() = %v, want non empty", b)
	// 			}
	// 		}()
	// 		wg.Done()
	// 	}
	// 	wg.Wait()
	// })
}

func TestMergeProfiles(t *testing.T) {
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "localhost:5000/gopher-pprof:latest",
			WaitingFor: wait.ForAll(
				wait.ForHTTP("/healthz").WithPort("8080"),
			),
		},
		Started: true,
		Logger:  testcontainers.TestLogger(t),
	})
	defer container.Terminate(ctx)

	collector := GoPprofCollector{}
	if err != nil {
		t.Fatal(err)
	}

	t.Run("check gettting pprof ", func(t *testing.T) {
		endpoint, err := container.Endpoint(ctx, "http")
		if err != nil {
			t.Fatal(err)
		}
		profile1 := bytes.Buffer{}
		profile2 := bytes.Buffer{}
		err = collector.GetProfile(fmt.Sprintf("%s/%s", endpoint, "debug/pprof/profile"), 1*time.Second, &profile1)
		if err != nil {
			t.Errorf("CollectPProf() error = %v", err)
			return
		}

		err = collector.GetProfile(fmt.Sprintf("%s/%s", endpoint, "debug/pprof/profile"), 2*time.Second, &profile2)
		if err != nil {
			t.Errorf("CollectPProf() error = %v", err)
			return
		}

		merged := bytes.Buffer{}
		err = collector.MergeProfiles([]io.Reader{&profile1, &profile2}, &merged)
		if err != nil {
			t.Errorf("MergeProfiles() error = %v", err)
			return
		}

		if len(merged.Bytes()) == 0 {
			t.Errorf("MergeProfiles() = %v, want non empty", merged.Bytes())
		}

		_, err = profile.Parse(&merged)
		if err != nil {
			t.Errorf("MergeProfiles() error = %v", err)
			return
		}

	})
}
