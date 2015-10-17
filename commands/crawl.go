package commands

import (
	"fmt"
	"github.com/andygrunwald/watson/client"
	"github.com/andygrunwald/watson/storage"
	"github.com/andygrunwald/watson/storage/identity"
	"github.com/codegangsta/cli"
	"log"
	"os"
	"sync"
)

// Crawl crawls a Gerrit instance
func Crawl(c *cli.Context) {
	watson, err := client.NewGerritClient(c.GlobalString("instance"), 60)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	concurrentNum := c.Int("concurrent")
	s := client.NewSemaphore(concurrentNum)

	watson.Authentication(c.String("auth-mode"), c.String("username"), c.String("password"))

	var wg sync.WaitGroup
	// Init storage
	storageChan, store, err := storage.GetStorage(c.String("storage"), &wg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	identityChan, identityStore, err  := identity.GetStorage(c.String("identity-storage"), &wg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func(){
		store.Close()
		identityStore.Close()
	}()

	crawl := client.NewCrawler(watson)
	crawl.ChangeSetQueryLimit = watson.GetQueryLimit()
	crawl.Storage = storageChan
	crawl.IdentityStorage = identityChan

	log.Println("Start crawling ...")

	projects, err := crawl.Projects()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for name := range *projects {
		wg.Add(1)
		s.Lock()
		log.Printf("Crawling project %s ...", name)

		go func(crawl *client.Crawler, name string) {
			defer func(){
				wg.Done()
				s.Unlock()
			}()
			crawl.Changesets(name)

			// * proceedChangeSetsDependsOnRelation
			// * proceedChangeSetsNeededByRelation
		}(crawl, name)
	}

	wg.Wait()
}
