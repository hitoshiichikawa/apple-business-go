// Package applebusiness provides the shared core for the Apple Business Go SDK:
// OAuth 2.0 (client_credentials) authentication with an ES256 JWT client
// assertion, an HTTP client with rate-limit-aware retries, JSON:API envelopes,
// and generic request helpers.
//
// Domain functionality is organized into sibling packages that operate on a
// *applebusiness.Client, mirroring the Apple Business platform pillars:
//
//	devices  - device management (org devices, MDM servers, activities, AppleCare)
//	people   - people management (users, user groups)
//	brand    - brand management (planned)
//	support  - support (planned)
//
// # Quick start
//
//	c, err := applebusiness.NewClient(applebusiness.Config{
//		Credentials: applebusiness.Credentials{
//			ClientID:   os.Getenv("AXM_CLIENT_ID"),
//			TeamID:     os.Getenv("AXM_TEAM_ID"),
//			KeyID:      os.Getenv("AXM_KEY_ID"),
//			PrivateKey: pemBytes,
//		},
//	})
//	devs, err := devices.New(c).List(ctx, nil)
//
// This is an unofficial SDK; verify behavior against Apple's official
// documentation before production use.
package applebusiness
