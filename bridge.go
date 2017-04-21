package geobin

import (
	"encoding/binary"

	"github.com/tidwall/tile38/geojson"
)

type BBox struct {
	Min, Max Position
}

// WithinBBox detects if the object is fully contained inside a bbox.
func (g Object) WithinBBox(bbox BBox) bool {
	return g.bridge().WithinBBox(geojson.BBox{
		Min: geojson.Position{bbox.Min.X, bbox.Min.Y, bbox.Min.Z},
		Max: geojson.Position{bbox.Max.X, bbox.Max.Y, bbox.Max.Z},
	})
}

// IntersectsBBox detects if the object intersects a bbox.
func (g Object) IntersectsBBox(bbox BBox) bool {
	return g.bridge().IntersectsBBox(geojson.BBox{
		Min: geojson.Position{bbox.Min.X, bbox.Min.Y, bbox.Min.Z},
		Max: geojson.Position{bbox.Max.X, bbox.Max.Y, bbox.Max.Z},
	})
}

// Within detects if the object is fully contained inside another object.
func (g Object) Within(o Object) bool {
	return g.bridge().Within(o.bridge())
}

// Intersects detects if the object intersects another object.
func (g Object) Intersects(o Object) bool {
	return g.bridge().Intersects(o.bridge())
}

// Nearby detects if the object is nearby a position.
func (g Object) Nearby(center Position, meters float64) bool {
	return g.bridge().Nearby(
		geojson.Position{center.X, center.Y, center.Z}, meters,
	)
}

// CalculatedBBox is exterior bbox containing the object.
func (g Object) CalculatedBBox() BBox {
	b := g.bridge().CalculatedBBox()
	return BBox{
		Min: Position{b.Min.X, b.Min.Y, b.Min.Z},
		Max: Position{b.Max.X, b.Max.Y, b.Max.Z},
	}
}

// CalculatedPoint is a point representation of the object.
func (g Object) CalculatedPoint() Position {
	p := g.bridge().CalculatedPoint()
	return Position{p.X, p.Y, p.Z}
}

// Geohash converts the object to a geohash value.
func (g Object) Geohash(precision int) (string, error) {
	return g.bridge().Geohash(precision)
}

// IsBBoxDefined returns true if the object has a defined bbox.
func (g Object) IsBBoxDefined() bool { return g.bridge().IsBBoxDefined() }

func (g Object) Sparse(amount byte) []Object {
	gb := g.BBox()
	b := geojson.BBox{
		Min: geojson.Position{gb.Min.X, gb.Min.Y, gb.Min.Z},
		Max: geojson.Position{gb.Max.X, gb.Max.Y, gb.Max.Z},
	}
	bb := b.Sparse(amount)
	var res []Object
	for _, b := range bb {
		res = append(res, Make3DRect(b.Min.X, b.Min.Y, b.Min.Z, b.Max.X, b.Max.Y, b.Max.Z))
	}
	return res
}

func BBoxFromCenter(lat float64, lon float64, meters float64) Object {
	b := geojson.BBoxesFromCenter(lat, lon, meters)
	return Make2DRect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y)
}

// DistanceTo calculates the distance to a position
func (p Position) DistanceTo(position Position) float64 {
	p1 := geojson.Position{p.X, p.Y, p.Z}
	p2 := geojson.Position{position.X, position.Y, position.Z}
	return p1.DistanceTo(p2)
}

// Destination calculates a new position based on the distance and bearing.
func (p Position) Destination(meters, bearingDegrees float64) Position {
	p1 := geojson.Position{p.X, p.Y, p.Z}
	p2 := p1.Destination(meters, bearingDegrees)
	return Position{p2.X, p2.Y, p2.Z}
}

func geomReadPosition(data []byte, dims int) (geojson.Position, []byte) {
	var p geojson.Position
	p.X, data = readFloat64(data)
	p.Y, data = readFloat64(data)
	if dims == 3 {
		p.Z, data = readFloat64(data)
	}
	return p, data
}

func geomReadBBox(data []byte, bboxSize int) geojson.BBox {
	switch bboxSize {
	case 48:
		var min, max geojson.Position
		min.X, data = readFloat64(data)
		min.Y, data = readFloat64(data)
		min.Z, data = readFloat64(data)
		max.X, data = readFloat64(data)
		max.Y, data = readFloat64(data)
		max.Z, data = readFloat64(data)
		return geojson.BBox{min, max}
	case 32:
		var min, max geojson.Position
		min.X, data = readFloat64(data)
		min.Y, data = readFloat64(data)
		max.X, data = readFloat64(data)
		max.Y, data = readFloat64(data)
		return geojson.BBox{min, max}
	case 24:
		p, _ := geomReadPosition(data, 3)
		return geojson.BBox{p, p}
	case 16:
		p, _ := geomReadPosition(data, 2)
		return geojson.BBox{p, p}
	}
	return geojson.BBox{}
}

func (o Object) bridge() geojson.Object {
	if len(o.data) == 0 {
		// empty geometry
		return geojson.String("")
	}
	tail := o.data[len(o.data)-1]
	if tail&1 == 0 {
		// object is a string
		return geojson.String(o.String())
	}
	var dims int
	var bboxSize int
	if tail>>1&1 == 1 {
		dims = 3
		if tail>>2&1 == 1 {
			// 3D rect
			bboxSize = 48
		} else {
			// 3D point
			bboxSize = 24
		}
	} else {
		dims = 2
		if tail>>2&1 == 1 {
			// 2D rect
			bboxSize = 32
		} else {
			// 2D point
			bboxSize = 16
		}
	}
	if (tail>>3)&1 == 0 {
		// simple
		switch bboxSize {
		default:
			return geojson.String("") // invalid
		case 48, 32:
			// simple rect, bbox around a center point
			bbox := geomReadBBox(o.data, bboxSize)
			return geojson.Point{
				Coordinates: geojson.Position{
					X: (bbox.Max.X + bbox.Min.X) / 2,
					Y: (bbox.Max.Y + bbox.Min.Y) / 2,
					Z: (bbox.Max.Z + bbox.Min.Z) / 2,
				},
				BBox: &bbox,
			}
		case 24:
			// simple 3d point
			p, _ := geomReadPosition(o.data, dims)
			return geojson.Point{Coordinates: p}
		case 16:
			// simple 2d point
			p, _ := geomReadPosition(o.data, dims)
			return geojson.SimplePoint{p.X, p.Y}
		}
	}
	var exsz int
	if tail>>4&1 == 1 {
		// has exdata, skip over
		exsz = int(binary.LittleEndian.Uint32(o.data[len(o.data)-5:]))
	}
	geomData := o.data[bboxSize+exsz:]
	geomHead := geomData[0]
	geomType := GeometryType(geomHead >> 4)
	geomData = geomData[1:]
	if geomHead&1 == 1 {
		// hasMembers
		sz := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4+sz:]
	}
	var bbox *geojson.BBox
	if geomHead>>1&1 == 1 {
		// export bbox
		v := geomReadBBox(o.data, bboxSize)
		bbox = &v
	}
	// complex, let's pull the geom data
	switch geomType {
	default:
		return geojson.String("")
	case Point:
		p, _ := geomReadPosition(geomData, dims)
		return geojson.Point{Coordinates: p, BBox: bbox}
	case MultiPoint, LineString:
		n := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4:]
		coords := make([]geojson.Position, n)
		for i := 0; i < n; i++ {
			coords[i], geomData = geomReadPosition(geomData, dims)
		}
		if geomType == MultiPoint {
			return geojson.MultiPoint{coords, bbox}
		}
		return geojson.LineString{coords, bbox}
	case MultiLineString, Polygon:
		n := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4:]
		coords := make([][]geojson.Position, n)
		for i := 0; i < n; i++ {
			nn := int(binary.LittleEndian.Uint32(geomData))
			geomData = geomData[4:]
			coords[i] = make([]geojson.Position, nn)
			for j := 0; j < nn; j++ {
				coords[i][j], geomData = geomReadPosition(geomData, dims)
			}
		}
		if geomType == MultiLineString {
			return geojson.MultiLineString{coords, bbox}
		}
		return geojson.Polygon{coords, bbox}
	case MultiPolygon:
		n := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4:]
		coords := make([][][]geojson.Position, n)
		for i := 0; i < n; i++ {
			nn := int(binary.LittleEndian.Uint32(geomData))
			geomData = geomData[4:]
			coords[i] = make([][]geojson.Position, nn)
			for j := 0; j < nn; j++ {
				nnn := int(binary.LittleEndian.Uint32(geomData))
				geomData = geomData[4:]
				coords[i][j] = make([]geojson.Position, nnn)
				for k := 0; k < nnn; k++ {
					coords[i][j][k], geomData = geomReadPosition(geomData, dims)
				}
			}
		}
		return geojson.MultiPolygon{coords, bbox}
	case GeometryCollection, FeatureCollection:
		n := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4:]
		objs := make([]geojson.Object, n)
		for i := 0; i < n; i++ {
			sz := int(binary.LittleEndian.Uint32(geomData))
			o := Object{geomData[4 : 4+sz : 4+sz]}
			geomData = geomData[4+sz:]
			objs[i] = o.bridge()
		}
		if geomType == GeometryCollection {
			return geojson.GeometryCollection{objs, bbox}
		}
		return geojson.FeatureCollection{objs, bbox}
	case Feature:
		sz := int(binary.LittleEndian.Uint32(geomData))
		o := Object{geomData[4 : 4+sz : 4+sz]}
		geom := o.bridge()
		return geojson.Feature{
			Geometry: geom,
			BBox:     bbox,
			//idprops:  string(o.Members()),
		}
	}
}
