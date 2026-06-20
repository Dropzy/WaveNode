package artistmeta

import (
	"context"
	"fmt"
	"os"
)

type Pipeline struct {
	MusicBrainz *MusicBrainzProvider
	Wikidata    *WikidataProvider
	Commons     *WikimediaCommonsProvider
	FanartTV    *FanartTvProvider
}

func NewPipeline(cache APIResponseCache) *Pipeline {
	return &Pipeline{
		MusicBrainz: NewMusicBrainzProvider(cache),
		Wikidata:    NewWikidataProvider(cache),
		Commons:     NewWikimediaCommonsProvider(cache),
		FanartTV:    &FanartTvProvider{APIKey: os.Getenv("FANART_TV_API_KEY")},
	}
}

func (p *Pipeline) Lookup(ctx context.Context, artistName string) (*LookupResult, error) {
	match, err := p.MusicBrainz.BestArtistMatch(ctx, artistName)
	if err != nil {
		return nil, err
	}

	candidates := make([]ImageCandidate, 0, 2)
	if match.WikidataID != "" {
		file, err := p.Wikidata.CommonsFileForEntity(ctx, match.WikidataID)
		if err == nil {
			candidate, err := p.Commons.ImageCandidateForFile(ctx, file, match.ConfidenceScore)
			if err == nil {
				candidates = append(candidates, *candidate)
			}
		}
	}

	result := &LookupResult{
		Artist:     *match,
		Candidates: candidates,
		Refreshed:  true,
	}
	if len(candidates) > 0 {
		result.Image = &candidates[0]
		return result, nil
	}
	return result, fmt.Errorf("no reusable artist image found")
}
