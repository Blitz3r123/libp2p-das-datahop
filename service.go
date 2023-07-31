package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	dht "github.com/Blitz3r123/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	// dht "./vendor/go-libp2p-kad-dht"
)

type Service struct {
	rpcServer *rpc.Server
	rpcClient *rpc.Client
	host      host.Host
	protocol  protocol.ID
}

type Parcel struct {
	StartingIndex int
	IsRow         bool
	SampleCount   int
}

func NewService(host host.Host, protocol protocol.ID) *Service {
	return &Service{
		host:     host,
		protocol: protocol,
	}
}

func (s *Service) SetupRPC() error {
	echoRPCAPI := EchoRPCAPI{service: s}

	s.rpcServer = rpc.NewServer(s.host, s.protocol)
	err := s.rpcServer.Register(&echoRPCAPI)
	if err != nil {
		return err
	}

	s.rpcClient = rpc.NewClientWithServer(s.host, s.protocol, s.rpcServer)
	return nil
}

func SplitSamplesIntoParcels(RowCount, ParcelSize int) []Parcel {
	TotalSamplesCount := RowCount * RowCount
	parcels := make([]Parcel, 0)

	// Split the samples into row parcels
	for i := 0; i < TotalSamplesCount; i += ParcelSize {
		parcel := Parcel{
			StartingIndex: i,
			SampleCount:   ParcelSize,
			IsRow:         true,
		}
		parcels = append(parcels, parcel)
	}

	// Split the samples into column parcels
	rowID := 0
	colID := 0
	for colID < RowCount {
		for i := 0; i < ParcelSize; i++ {
			parcelID := rowID*RowCount + colID
			parcel := Parcel{
				StartingIndex: parcelID,
				SampleCount:   ParcelSize,
				IsRow:         false,
			}
			if i == 0 {
				parcels = append(parcels, parcel)
			}

			rowID++

			if rowID >= RowCount {
				rowID = 0
				colID++
			}
		}
	}

	return parcels
}

func (s *Service) StartMessaging(dht *dht.IpfsDHT, stats *Stats, peerType string, ctx context.Context) {
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()

	const RowCount = 4
	const TotalSamplesCount = RowCount * RowCount
	const TotalBlocksCount = 10
	const ParcelSize = 2
	blockID := 0
	currentBlockID := 0

	// var sample []byte = make([]byte, 512)
	var samplesReceived []int

	// ? Generate 512 x 512 IDs for each sample
	sampleIDs := make([]int, TotalSamplesCount)
	for i := 0; i < TotalSamplesCount; i++ {
		sampleIDs[i] = i
	}

	parcels := SplitSamplesIntoParcels(RowCount, ParcelSize)
	var parcelsSent []Parcel
	// var parcelsReceived []Parcel

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			if peerType == "builder" {

				// ? If all parcels are sent, go to the next block
				if len(parcelsSent) == len(parcels) && blockID < TotalBlocksCount {
					blockID += 1
					parcelsSent = make([]Parcel, 0)
				}

				// ? If all blocks are sent, stop
				if blockID >= TotalBlocksCount {
					fmt.Println("All blocks sent.")
					continue
				}

				// ? Pick a random parcel that has not been sent yet
				parcelID := rand.Intn(len(parcels))
				for _, p := range parcelsSent {
					// ? If the parcel ID and type are the same, pick another parcel
					if p.StartingIndex == parcels[parcelID].StartingIndex && p.IsRow == parcels[parcelID].IsRow {
						continue
					}
				}

				// ? Get the parcel
				parcelToSend := parcels[parcelID]

				// ? Get the samples
				parcelSamplesToSend := make([]byte, parcelToSend.SampleCount)

				peers := FilterSelf(s.host.Peerstore().Peers(), s.host.ID())
				dhtPeers := FilterSelf(dht.RoutingTable().ListPeers(), s.host.ID())

				if len(peers) == 0 && len(dhtPeers) == 0 {
					continue
				}

				if len(peers) == 0 || len(dhtPeers) == 0 {
					for _, p := range peers {
						_, err := dht.RoutingTable().TryAddPeer(p, false, true)
						if err != nil {
							log.Printf("Failed to add peer %s : %s\n", p[0:5].Pretty(), err.Error())
						}
					}
				}

				startTime := time.Now()

				// ? Put parcel samples into DHT
				putErr := dht.PutValue(ctx, "/das/sample/"+fmt.Sprint(blockID)+"/"+fmt.Sprint(parcelToSend.StartingIndex), parcelSamplesToSend)

				if putErr != nil {
					log.Print("[BUILDER\t\t" + s.host.ID()[0:5].Pretty() + "] PutValue() Error: " + putErr.Error())
					log.Printf("[BUILDER\t\t"+s.host.ID()[0:5].Pretty()+"]: DHT Peers: %d\n", len(dht.RoutingTable().ListPeers()))
					stats.TotalFailedPuts += 1
					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
				} else {
					parcelsSent = append(parcelsSent, parcelToSend)

					parcelTypeString := colorize("COL", "green")
					lastParcelID := parcelToSend.StartingIndex + (parcelToSend.SampleCount-1)*RowCount
					if parcelToSend.IsRow {
						parcelTypeString = colorize("ROW", "blue")
						lastParcelID = parcelToSend.StartingIndex + (parcelToSend.SampleCount - 1)
					}

					builderIdString := "[BUILDER\t" + s.host.ID()[0:5].Pretty() + "]"
					blockIDString := fmt.Sprint(blockID)

					parcelContentsString := "[" + strconv.Itoa(parcelToSend.StartingIndex) + "..." + strconv.Itoa(lastParcelID) + "]"
					if parcelToSend.SampleCount <= 2 {
						parcelContentsString = "[" + strconv.Itoa(parcelToSend.StartingIndex) + ", " + strconv.Itoa(lastParcelID) + "]"
					}

					log.Print(builderIdString + " PUT " + parcelTypeString + " parcel BID " + blockIDString + " into DHT:\t" + parcelContentsString + "\n")

					stats.TotalPutMessages += 1
					stats.PutLatencies = append(stats.PutLatencies, time.Since(startTime))
				}

			} else if peerType == "validator" {
				randomSampleID := rand.Intn(TotalSamplesCount)

				startTime := time.Now()
				_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(currentBlockID)+"/"+fmt.Sprint(randomSampleID))
				if err != nil {
					// log.Print("[VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] GetValue(/das/sample/" + fmt.Sprint(currentBlockID) + "/" + fmt.Sprint(randomSampleID) + ") Error: " + err.Error())
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)
				} else {
					log.Print("[VALIDATOR\t\t" + s.host.ID()[0:5].Pretty() + "] " + colorize("GET", "blue") + " sample (" + fmt.Sprint(blockID) + ", " + fmt.Sprint(randomSampleID) + ") from DHT.\n")
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)

					samplesReceived = append(samplesReceived, randomSampleID)
				}

				validatorRequiredSampleCount := 4*512 - 4 // ? 2 columns and 2 rows of samples (take away the 4 intersecting samples)

				if len(samplesReceived) == validatorRequiredSampleCount+75 { // ? 75 random samples
					currentBlockID += 1
					samplesReceived = make([]int, 0)
				}

			} else if peerType == "nonvalidator" {
				randomSampleID := rand.Intn(TotalSamplesCount)

				startTime := time.Now()

				_, hops, err := dht.GetValueHops(ctx, "/das/sample/"+fmt.Sprint(currentBlockID)+"/"+fmt.Sprint(randomSampleID))
				if err != nil {
					// log.Print("[NON VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] GetValue(/das/sample/" + fmt.Sprint(currentBlockID) + "/" + fmt.Sprint(randomSampleID) + ") Error: " + err.Error())
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)
				} else {
					log.Print("[NON VALIDATOR\t" + s.host.ID()[0:5].Pretty() + "] " + colorize("GET", "blue") + " sample (" + fmt.Sprint(blockID) + ", " + fmt.Sprint(randomSampleID) + ") from DHT.\n")
					stats.TotalGetMessages += 1
					stats.GetLatencies = append(stats.GetLatencies, time.Since(startTime))
					stats.GetHops = append(stats.GetHops, hops)

					samplesReceived = append(samplesReceived, randomSampleID)
				}

				if len(samplesReceived) == 75 { // ? 75 random samples
					currentBlockID += 1
					samplesReceived = make([]int, 0)
				}

			}

		}
	}
}

func (s *Service) ReceiveEcho(envelope Envelope) Envelope {
	fmt.Printf("Peer %s got 42KB\n", s.host.ID())

	return Envelope{
		Message: fmt.Sprintf("Peer %s got 42KB", s.host.ID()),
	}
}

func FilterSelf(peers peer.IDSlice, self peer.ID) peer.IDSlice {
	var withoutSelf peer.IDSlice
	for _, p := range peers {
		if p != self {
			withoutSelf = append(withoutSelf, p)
		}
	}
	return withoutSelf
}

func Ctxts(n int) []context.Context {
	ctxs := make([]context.Context, n)
	for i := 0; i < n; i++ {
		ctxs[i] = context.Background()
	}
	return ctxs
}

func CopyEnvelopesToIfaces(in []*Envelope) []interface{} {
	ifaces := make([]interface{}, len(in))
	for i := range in {
		in[i] = &Envelope{}
		ifaces[i] = in[i]
	}
	return ifaces
}
