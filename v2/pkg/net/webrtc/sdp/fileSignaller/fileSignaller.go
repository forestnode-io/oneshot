package filesignaller

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

type fileSignaller struct {
	dirPath string
	watcher *fsnotify.Watcher
}

func New(dir string) sdp.Signaller {
	return &fileSignaller{
		dirPath: dir,
	}
}

func (s *fileSignaller) Start(ctx context.Context, handler sdp.RequestHandler) error {
	err := os.RemoveAll(s.dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		} else {
			return fmt.Errorf("unable to remove directory: %w", err)
		}
	}
	if err = os.Mkdir(s.dirPath, 0755); err != nil {
		return fmt.Errorf("unable to create directory: %w", err)
	}

	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("unable to create file watcher: %w", err)
	}

	go func() {
		for {
			select {
			case event := <-s.watcher.Events:
				fmt.Printf("event: %+v\n", event)
				if event.Has(fsnotify.Create) {
					stat, err := os.Stat(event.Name)
					if err != nil {
						log.Println("unable to stat file:", err)
						return
					}
					if !stat.IsDir() {
						continue
					}
				}

				handler.HandleRequest(ctx, func(ctx context.Context, o sdp.Offer) (sdp.Answer, error) {
					offerPath := filepath.Join(event.Name, "offer")
					offer, err := o.JSON()
					if err != nil {
						return "", fmt.Errorf("unable to marshal offer: %w", err)
					}
					if err := os.WriteFile(offerPath, offer, 0755); err != nil {
						return "", fmt.Errorf("unable to write offer: %w", err)
					}

					watcher, err := fsnotify.NewWatcher()
					if err != nil {
						return "", fmt.Errorf("unable to create file watcher: %w", err)
					}
					defer watcher.Close()

					answerChan := make(chan sdp.Answer)
					go func() {
						answerPath := filepath.Join(event.Name, "answer")
						for {
							select {
							case event := <-watcher.Events:
								if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
									if event.Name != answerPath {
										continue
									}
									answerBytes, err := os.ReadFile(answerPath)
									if err != nil {
										log.Printf("unable to read answer: %v", err)
										return
									}

									if len(answerBytes) != 0 {
										answer, err := sdp.AnswerFromJSON(answerBytes)
										if err != nil {
											log.Printf("unable to unmarshal answer: %v", err)
											return
										}
										answerChan <- answer
										return
									}
								}
							case err := <-watcher.Errors:
								if err != nil {
									log.Printf("error from answer watcher: %v", err)
									return
								}
							}
						}
					}()
					watcher.Add(event.Name)

					ans := <-answerChan
					return ans, nil
				})
			case err := <-s.watcher.Errors:
				if err != nil {
					log.Printf("error from sdp dir watcher: %v", err)
				}
				return
			}
		}
	}()
	s.watcher.Add(s.dirPath)

	if err = os.Mkdir(filepath.Join(s.dirPath, "0"), 0755); err != nil {
		return fmt.Errorf("unable to create dir: %w", err)
	}

	<-ctx.Done()
	return nil
}

func (s *fileSignaller) Shutdown() error {
	s.watcher.Close()
	return nil
}
