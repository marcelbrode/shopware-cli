package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"shopware-cli/extension"
	"strings"

	"github.com/pkg/errors"

	termColor "github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var accountCompanyProducerExtensionInfoPullCmd = &cobra.Command{
	Use:   "pull [path]",
	Short: "Generates local store configuration from account data",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getAccountAPIByConfig()

		path, err := filepath.Abs(args[0])

		if err != nil {
			log.Fatalln(fmt.Errorf("pull: %v", err))
		}

		zipExt, err := extension.GetExtensionByFolder(path)

		if err != nil {
			log.Fatalln(fmt.Errorf("pull: %v", err))
		}

		zipName, err := zipExt.GetName()

		if err != nil {
			log.Fatalln(fmt.Errorf("pull: %v", err))
		}

		p, err := client.Producer()

		if err != nil {
			log.Fatalln(fmt.Errorf("pull: %v", err))
		}

		storeExt, err := p.GetExtensionByName(zipName)

		if err != nil {
			log.Fatalln(fmt.Errorf("pull: %v", err))
		}

		resourcesFolder := fmt.Sprintf("%s/src/Resources/store/", zipExt.GetPath())
		categoryList := make([]string, 0)
		availabilities := make([]string, 0)
		localizations := make([]string, 0)
		tagsDE := make([]string, 0)
		tagsEN := make([]string, 0)
		videosDE := make([]string, 0)
		videosEN := make([]string, 0)
		highlightsDE := make([]string, 0)
		highlightsEN := make([]string, 0)
		featuresDE := make([]string, 0)
		featuresEN := make([]string, 0)
		faqDE := make([]extension.ConfigStoreFaq, 0)
		faqEN := make([]extension.ConfigStoreFaq, 0)
		images := make([]extension.ConfigStoreImage, 0)

		if _, err := os.Stat(resourcesFolder); os.IsNotExist(err) {
			err = os.MkdirAll(resourcesFolder, os.ModePerm)

			if err != nil {
				log.Fatalln(fmt.Errorf("pull: %v", err))
			}
		}

		var iconConfigPath *string

		if len(storeExt.IconURL) > 0 {
			icon := "src/Resources/store/icon.png"
			iconConfigPath = &icon
			err := downloadFileTo(storeExt.IconURL, fmt.Sprintf("%s/icon.png", resourcesFolder))
			if err != nil {
				log.Fatalln(err)
			}
		}

		for _, category := range storeExt.Categories {
			categoryList = append(categoryList, category.Name)
		}

		for _, localization := range storeExt.Localizations {
			localizations = append(localizations, localization.Name)
		}

		for _, a := range storeExt.StoreAvailabilities {
			availabilities = append(availabilities, a.Name)
		}

		storeImages, err := p.GetExtensionImages(storeExt.Id)

		if err != nil {
			log.Fatalln(fmt.Errorf("pull: %v", err))
		}

		for i, image := range storeImages {
			imagePath := fmt.Sprintf("src/Resources/store/img-%d.png", i)
			err := downloadFileTo(image.RemoteLink, fmt.Sprintf("%s/%s", zipExt.GetPath(), imagePath))
			if err != nil {
				log.Fatalln(err)
			}

			images = append(images, extension.ConfigStoreImage{
				File:     imagePath,
				Preview:  extension.ConfigStoreImagePreview{German: image.Details[0].Preview, English: image.Details[1].Preview},
				Activate: extension.ConfigStoreImageActivate{German: image.Details[0].Activated, English: image.Details[1].Activated},
				Priority: image.Priority,
			})
		}

		for _, info := range storeExt.Infos {
			language := info.Locale.Name[0:2]

			if language == "de" {
				for _, element := range info.Tags {
					tagsDE = append(tagsDE, element.Name)
				}

				for _, element := range info.Videos {
					videosDE = append(videosDE, element.URL)
				}

				highlightsDE = append(highlightsDE, strings.Split(info.Highlights, "\n")...)
				featuresDE = append(featuresDE, strings.Split(info.Features, "\n")...)

				for _, element := range info.Faqs {
					faqDE = append(faqDE, extension.ConfigStoreFaq{Question: element.Question, Answer: element.Answer})
				}
			} else {
				for _, element := range info.Tags {
					tagsEN = append(tagsEN, element.Name)
				}

				for _, element := range info.Videos {
					videosEN = append(videosEN, element.URL)
				}

				highlightsEN = append(highlightsEN, strings.Split(info.Highlights, "\n")...)
				featuresEN = append(featuresEN, strings.Split(info.Features, "\n")...)

				for _, element := range info.Faqs {
					faqEN = append(faqEN, extension.ConfigStoreFaq{Question: element.Question, Answer: element.Answer})
				}
			}
		}

		newCfg := extension.Config{Store: extension.ConfigStore{
			Icon:                                iconConfigPath,
			DefaultLocale:                       &storeExt.StandardLocale.Name,
			Type:                                &storeExt.ProductType.Name,
			AutomaticBugfixVersionCompatibility: &storeExt.AutomaticBugfixVersionCompatibility,
			Availabilities:                      &availabilities,
			Localizations:                       &localizations,
			Description:                         extension.ConfigTranslatedString{German: &storeExt.Infos[0].Description, English: &storeExt.Infos[1].Description},
			InstallationManual:                  extension.ConfigTranslatedString{German: &storeExt.Infos[0].InstallationManual, English: &storeExt.Infos[1].InstallationManual},
			Categories:                          &categoryList,
			Tags:                                extension.ConfigTranslatedStringList{German: &tagsDE, English: &tagsEN},
			Videos:                              extension.ConfigTranslatedStringList{German: &videosDE, English: &videosEN},
			Highlights:                          extension.ConfigTranslatedStringList{German: &highlightsDE, English: &highlightsEN},
			Features:                            extension.ConfigTranslatedStringList{German: &featuresDE, English: &featuresEN},
			Faq:                                 extension.ConfigStoreTranslatedFaq{German: &faqDE, English: &faqEN},
			Images:                              &images,
		}}

		content, err := yaml.Marshal(newCfg)

		if err != nil {
			log.Fatalln(fmt.Errorf("pull: %v", err))
		}

		extCfgFile := fmt.Sprintf("%s/%s", zipExt.GetPath(), ".shopware-extension.yml")
		err = ioutil.WriteFile(extCfgFile, content, os.ModePerm)

		if err != nil {
			log.Fatalln(fmt.Errorf("pull: %v", err))
		}

		termColor.Green("Files has been written to the given extension folder")
	},
}

func init() {
	accountCompanyProducerExtensionInfoCmd.AddCommand(accountCompanyProducerExtensionInfoPullCmd)
}

func downloadFileTo(url string, target string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx
	if err != nil {
		return errors.Wrap(err, "create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "download file")
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read file body")
	}

	err = ioutil.WriteFile(target, content, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "write to file")
	}

	return nil
}