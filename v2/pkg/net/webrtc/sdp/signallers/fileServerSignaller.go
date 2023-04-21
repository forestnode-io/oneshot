package signallers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fsnotify/fsnotify"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

type fileServerSignaller struct {
	dirPath string
	watcher *fsnotify.Watcher
	config  *webrtc.Configuration
}

func NewFileServerSignaller(dir string, config *webrtc.Configuration) ServerSignaller {
	return &fileServerSignaller{
		dirPath: dir,
		config:  config,
	}
}

func (s *fileServerSignaller) Start(ctx context.Context, handler RequestHandler) error {
	log := zerolog.Ctx(ctx)
	stat, err := os.Stat(s.dirPath)
	if err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("path is not a directory")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("unable to stat directory: %w", err)
	}

	err = os.RemoveAll(s.dirPath)
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
		id := 0
		for {
			select {
			case event := <-s.watcher.Events:
				if event.Has(fsnotify.Create) {
					stat, err := os.Stat(event.Name)
					if err != nil {
						log.Error().Err(err).
							Msg("unable to stat file")
						return
					}
					if !stat.IsDir() {
						continue
					}
				}

				handler.HandleRequest(ctx, strconv.Itoa(id), s.config, func(ctx context.Context, id string, o sdp.Offer) (sdp.Answer, error) {
					offerPath := filepath.Join(event.Name, "offer")
					offer, err := o.MarshalJSON()
					if err != nil {
						return "", fmt.Errorf("unable to marshal offer: %w", err)
					}
					if err := os.WriteFile(offerPath, offer, 0755); err != nil {
						return "", fmt.Errorf("unable to write offer: %w", err)
					}
					file, err := os.Create(filepath.Join(event.Name, "answer"))
					if err != nil {
						return "", fmt.Errorf("unable to create answer file: %w", err)
					}
					file.Close()

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
										log.Error().Err(err).
											Msg("unable to read answer file")
										return
									}

									if len(answerBytes) != 0 {
										answer, err := sdp.AnswerFromJSON(answerBytes)
										if err != nil {
											log.Error().Err(err).
												Msg("unable to unmarshal answer")
											return
										}
										answerChan <- answer
										return
									}
								}
							case err := <-watcher.Errors:
								if err != nil {
									log.Error().Err(err).
										Msg("error from answer watcher")
									return
								}
							}
						}
					}()
					watcher.Add(event.Name)

					ans := <-answerChan
					return ans, nil
				})

				id++
			case err := <-s.watcher.Errors:
				if err != nil {
					log.Error().Err(err).
						Msg("error from sdp dir watcher")
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

func (s *fileServerSignaller) Shutdown() error {
	s.watcher.Close()
	return nil
}
