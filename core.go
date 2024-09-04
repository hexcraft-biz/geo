package geo

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"

	geojson "github.com/paulmach/go.geojson"
)

const (
	degToKm       = 111.0
	earthRadiusKm = 6371.0
)

type Point struct {
	*geojson.Geometry
}

func Parse(longitude, latitude float64) Point {
	return Point{
		Geometry: geojson.NewPointGeometry([]float64{longitude, latitude}),
	}
}

func (p Point) StraightLineDistance(tp Point) float64 {
	if p.Geometry == nil || tp.Geometry == nil {
		return 0
	}

	lat1, lon1 := p.Point[1], p.Point[0]
	lat2, lon2 := tp.Point[1], tp.Point[0]

	dx := lon2 - lon1
	dy := lat2 - lat1

	distanceKm := math.Sqrt(dx*dx+dy*dy) * degToKm

	return distanceKm
}

func (p Point) Distance(tp Point) float64 {
	if p.Geometry == nil || tp.Geometry == nil {
		return 0
	}

	lat1, lon1 := p.Point[1], p.Point[0]
	lat2, lon2 := tp.Point[1], tp.Point[0]

	lat1Rad := toRadians(lat1)
	lon1Rad := toRadians(lon1)
	lat2Rad := toRadians(lat2)
	lon2Rad := toRadians(lon2)

	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadiusKm * c

	return distance
}

func (g *Point) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan Point: invalid type")
	}

	if len(b) != 25 || b[0] != 0 || b[1] != 1 {
		return errors.New("failed to scan Point: invalid POINT data")
	}

	lon := binary.LittleEndian.Uint64(b[9:17])
	lat := binary.LittleEndian.Uint64(b[17:25])

	g.Geometry = geojson.NewPointGeometry([]float64{
		float64FromBytes(lon),
		float64FromBytes(lat),
	})

	return nil
}

func (g Point) Value() (driver.Value, error) {
	if g.Geometry == nil || g.Type != "Point" {
		return nil, errors.New("invalid Point value")
	}

	lon := float64ToBytes(g.Point[0])
	lat := float64ToBytes(g.Point[1])

	data := make([]byte, 25)
	data[0] = 0
	data[1] = 1
	copy(data[9:17], lon)
	copy(data[17:25], lat)

	return data, nil
}

func (p Point) MarshalJSON() ([]byte, error) {
	if p.Geometry == nil || p.Type != "Point" {
		return nil, errors.New("invalid GeoJSON Point")
	}
	return json.Marshal(p.Geometry)
}

func (p *Point) UnmarshalJSON(data []byte) error {
	var g geojson.Geometry
	if err := json.Unmarshal(data, &g); err != nil {
		return err
	}

	if g.Type != "Point" || len(g.Point) != 2 {
		return errors.New("invalid GeoJSON Point")
	}

	p.Geometry = &g
	return nil
}

func float64FromBytes(bits uint64) float64 {
	return float64(int64(bits))
}

func float64ToBytes(f float64) []byte {
	bits := uint64(f)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, bits)
	return b
}

func toRadians(deg float64) float64 {
	return deg * (math.Pi / 180)
}
