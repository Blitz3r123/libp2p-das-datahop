package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
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

func SplitSamplesIntoParcels(RowCount, parcelSize int, parcelType string) []Parcel {
	TotalSamplesCount := RowCount * RowCount
	parcels := make([]Parcel, 0)

	// Split the samples into row parcels
	for i := 0; i < TotalSamplesCount; i += parcelSize {
		parcel := Parcel{
			StartingIndex: i,
			SampleCount:   parcelSize,
			IsRow:         true,
		}
		parcels = append(parcels, parcel)
	}

	// Split the samples into column parcels
	rowID := 0
	colID := 0
	for colID < RowCount {
		for i := 0; i < parcelSize; i++ {
			parcelID := rowID*RowCount + colID
			parcel := Parcel{
				StartingIndex: parcelID,
				SampleCount:   parcelSize,
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

	if parcelType == "all" {

		return parcels

	} else if parcelType == "row" {

		rowParcels := make([]Parcel, 0)
		for _, parcel := range parcels {
			if parcel.IsRow {
				rowParcels = append(rowParcels, parcel)
			}
		}
		return rowParcels

	} else if parcelType == "col" {

		colParcels := make([]Parcel, 0)
		for _, parcel := range parcels {
			if !parcel.IsRow {
				colParcels = append(colParcels, parcel)
			}
		}
		return colParcels

	}

	return parcels

}

func getParcelCounts(parcels []Parcel) (int, int) {
	rowParcelsCount := 0
	colParcelsCount := 0

	for _, parcel := range parcels {
		if parcel.IsRow {
			rowParcelsCount++
		} else {
			colParcelsCount++
		}
	}

	return rowParcelsCount, colParcelsCount
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

func (s *Service) StartMessaging(dht *dht.IpfsDHT, stats *Stats, peerType string, parcelSize int, ctx context.Context) {

	const RowCount = 512
	const TotalBlocksCount = 3

	if peerType == "builder" {

		blockID := 0
		for blockID < TotalBlocksCount {

			log.Printf("[B - %s] Starting to seed block %d...\n", s.host.ID()[0:5].Pretty(), blockID)

			// ! Seeding

			seedingTime := time.Now()

			parcels := SplitSamplesIntoParcels(RowCount, parcelSize, "all")

			// ? Timeout after a minute
			// ? There are cases where all recipients have received all they need and leave the DHT (execution stops) - so there are no more peers in the DHT
			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			for len(parcels) > 0 {

				if ctx.Err() != nil {
					log.Printf("[B - B%d] Seeding timed out after %.2f seconds.\n", blockID, time.Since(seedingTime).Seconds())

					if blockID == TotalBlocksCount-1 {
						return
					} else {
						break
					}
				}

				randomIndex := 0
				if len(parcels) > 1 {
					randomIndex = rand.Intn(len(parcels) - 1)
				}

				randomParcel := parcels[randomIndex]

				parcelSamplesToSend := make([]byte, randomParcel.SampleCount*512)

				parcelType := "row"
				if !randomParcel.IsRow {
					parcelType = "col"
				}

				putStartTime := time.Now()
				putErr := dht.PutValue(
					ctx,
					"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(randomParcel.StartingIndex),
					parcelSamplesToSend,
				)

				if putErr != nil {
					stats.TotalFailedPuts += 1
					stats.TotalPutMessages += 1
				} else {
					stats.PutLatencies = append(stats.PutLatencies, time.Since(putStartTime))
					stats.TotalPutMessages += 1

					parcels = append(parcels[:randomIndex], parcels[randomIndex+1:]...)

				}

				if len(parcels) == 0 {
					stats.SeedingLatencies = append(stats.SeedingLatencies, time.Since(seedingTime))
					log.Printf("[B - B%d] Seeding took %.2f seconds.\n", blockID, time.Since(seedingTime).Seconds())
				}

			}

			blockID += 1
		}

		log.Printf("[B - %s] Finished seeding all blocks.\n", s.host.ID()[0:5].Pretty())

	} else if peerType == "validator" {

		blockID := 0

		for blockID < TotalBlocksCount {

			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			log.Printf("[V - %s] Starting to sample block %d...\n", s.host.ID()[0:5].Pretty(), blockID)

			startTime := time.Now()
			rowSamplingDurationMicrosec := int64(0)
			colSamplingDurationMicrosec := int64(0)

			rowColParcelsNeededCount := (RowCount / 2) / parcelSize
			randomParcelsNeededCount := 75

			allParcels := SplitSamplesIntoParcels(RowCount, parcelSize, "all")
			rowParcels := SplitSamplesIntoParcels(RowCount, parcelSize, "row")
			colParcels := SplitSamplesIntoParcels(RowCount, parcelSize, "col")

			randomRowParcels := pickRandomParcels(rowParcels, rowColParcelsNeededCount)
			randomColParcels := pickRandomParcels(colParcels, rowColParcelsNeededCount)
			randomParcels := pickRandomParcels(allParcels, randomParcelsNeededCount)

			allRandomParcels := append(randomRowParcels, randomColParcels...)
			allRandomParcels = append(allRandomParcels, randomParcels...)

			for len(allRandomParcels) > 0 {

				if ctx.Err() != nil {
					log.Printf("[V - B%d] Sampling timed out after %.2f seconds.\n", blockID, time.Since(startTime).Seconds())
					break
				}

				randomIndex := 0
				if len(allRandomParcels) > 1 {
					randomIndex = rand.Intn(len(allRandomParcels))
				}

				randomParcel := allRandomParcels[randomIndex]

				parcelType := "row"
				if !randomParcel.IsRow {
					parcelType = "col"
				}

				getStartTime := time.Now()
				_, hops, err := dht.GetValueHops(
					ctx,
					"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(randomParcel.StartingIndex),
				)

				if err != nil {
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
				} else {
					stats.GetLatencies = append(stats.GetLatencies, time.Since(getStartTime))
					stats.TotalGetMessages += 1
					stats.TotalSuccessGets += 1
					stats.GetHops = append(stats.GetHops, hops)

					allRandomParcels = append(allRandomParcels[:randomIndex], allRandomParcels[randomIndex+1:]...)

					rowParcelCount, colParcelCount := getParcelCounts(allRandomParcels)

					if rowParcelCount == 0 && rowSamplingDurationMicrosec == 0 {
						rowSamplingDurationMicrosec = time.Since(startTime).Microseconds()
						stats.RowSamplingLatencies = append(stats.RowSamplingLatencies, time.Since(startTime))
					}

					if colParcelCount == 0 && colSamplingDurationMicrosec == 0 {
						colSamplingDurationMicrosec = time.Since(startTime).Microseconds()
						stats.ColSamplingLatencies = append(stats.ColSamplingLatencies, time.Since(startTime))
					}

				}

			}

			if len(allRandomParcels) == 0 {
				log.Printf("[V - B%d] All Sampling took %.2f seconds.\n", blockID, time.Since(startTime).Seconds())

				blockID += 1
			}
		}

	} else if peerType == "nonvalidator" {

		blockID := 0

		for blockID < TotalBlocksCount {

			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			log.Printf("[R - %s] Starting to sample block %d...\n", s.host.ID()[0:5].Pretty(), blockID)

			startTime := time.Now()
			randomSamplingDurationMicrosec := int64(0)

			randomParcelsNeededCount := 75

			allParcels := SplitSamplesIntoParcels(RowCount, parcelSize, "all")

			randomParcels := pickRandomParcels(allParcels, randomParcelsNeededCount)

			for len(randomParcels) > 0 {

				if ctx.Err() != nil {
					log.Printf("[R - B%d] Sampling timed out after %.2f seconds.\n", blockID, time.Since(startTime).Seconds())
					break
				}

				randomIndex := 0
				if len(randomParcels) > 1 {
					randomIndex = rand.Intn(len(randomParcels))
				}

				randomParcel := randomParcels[randomIndex]

				parcelType := "row"
				if !randomParcel.IsRow {
					parcelType = "col"
				}

				getStartTime := time.Now()
				_, hops, err := dht.GetValueHops(
					ctx,
					"/das/sample/"+fmt.Sprint(blockID)+"/"+parcelType+"/"+fmt.Sprint(randomParcel.StartingIndex),
				)

				if err != nil {
					stats.TotalFailedGets += 1
					stats.TotalGetMessages += 1
				} else {
					stats.GetLatencies = append(stats.GetLatencies, time.Since(getStartTime))
					stats.TotalGetMessages += 1
					stats.TotalSuccessGets += 1
					stats.GetHops = append(stats.GetHops, hops)

					randomParcels = append(randomParcels[:randomIndex], randomParcels[randomIndex+1:]...)

					if len(randomParcels) == 0 && randomSamplingDurationMicrosec == 0 {
						randomSamplingDurationMicrosec = time.Since(startTime).Microseconds()
						stats.RandomSamplingLatencies = append(stats.RandomSamplingLatencies, time.Since(startTime))
					}

				}

			}

			if len(randomParcels) == 0 {
				log.Printf("[R - B%d] All Sampling took %.2f seconds.\n", blockID, time.Since(startTime).Seconds())
				blockID += 1
			}
		}

	} else {
		panic("Peer type not recognized: " + peerType)
	}
}

func pickRandomParcels(parcels []Parcel, requiredCount int) []Parcel {
	randomParcels := make([]Parcel, 0)
	for i := 0; i < requiredCount; i++ {
		randomIndex := rand.Intn(len(parcels))
		randomParcel := parcels[randomIndex]

		// ? Check if the random parcel has already been picked
		alreadyPicked := false
		for _, p := range randomParcels {
			if p.StartingIndex == randomParcel.StartingIndex && p.IsRow == randomParcel.IsRow {
				alreadyPicked = true
				break
			}
		}

		// ? If the random parcel has not been picked, add it to the list
		if !alreadyPicked {
			randomParcels = append(randomParcels, randomParcel)
		}
	}

	return randomParcels
}
