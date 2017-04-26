package geobin

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"strconv"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
)

type Position struct {
	X, Y, Z float64
}

// Object represents a packed geobin object
type Object struct {
	data []byte
}

// ParseJSON parses GeoJSON and returns an geobin object.
func ParseJSON(json string) Object {
	return objectFromJSON(json)
}

// WrapBinary creates an object by wrapping data.
func WrapBinary(data []byte) Object {
	return Object{data}
}

// Make2DPoint returns a simple 2D point object.
func Make2DPoint(x, y float64) Object {
	return makeSixAny(1, 2, [6]float64{x, y, 0, 0, 0, 0})
}

// Make3DPoint returns a simple 3D point object.
func Make3DPoint(x, y, z float64) Object {
	return makeSixAny(3, 3, [6]float64{x, y, z, 0, 0, 0})
}

// Make2DRect returns a simple 2D point object.
func Make2DRect(minX, minY, maxX, maxY float64) Object {
	return makeSixAny(5, 4, [6]float64{minX, minY, maxX, maxY, 0, 0})
}

// Make3DRect returns a simple 3D rect object.
func Make3DRect(minX, minY, minZ, maxX, maxY, maxZ float64) Object {
	return makeSixAny(7, 6, [6]float64{minX, minY, minZ, maxX, maxY, maxZ})
}

// MakeString returns a non-geometry string object.
func MakeString(str string) Object {
	data := make([]byte, len(str)+1)
	copy(data, str)
	return Object{data}
}

// IsGeometry returns true if the object is a geometry.
func (o Object) IsGeometry() bool {
	return len(o.data) > 0 && o.data[len(o.data)-1]&1 == 1
}

// String returns a string representation of the object.
func (o Object) String() string {
	return string(o.AppendString(nil))
}

func (o Object) AppendString(b []byte) []byte {
	if o.IsGeometry() {
		return appendGeojsonBytes(b, o)
	}
	if len(o.data) == 0 {
		return b
	}
	if o.data[len(o.data)-1] == 0 {
		return append(b, o.data[:len(o.data)-1]...)
	}
	return append(b, o.parseComponents().data...)
}

// JSON returns a JSON representation of the object. Geometries are converted
// to GeoJSON and strings are simple JSON strings.
func (o Object) JSON() string {
	return string(o.AppendJSON(nil))
}

func (o Object) AppendJSON(b []byte) []byte {
	if o.IsGeometry() {
		return appendGeojsonBytes(b, o)
	}
	if len(o.data) == 0 {
		return append(b, "null"...)
	}
	var str []byte
	if o.data[len(o.data)-1] == 0 {
		str = o.data[:len(o.data)-1]
	} else {
		str = o.parseComponents().data
	}
	return appendJSONStringBytes(b, str)
}

func appendJSONStringBytes(b []byte, s []byte) []byte {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] == '\\' || s[i] == '"' || s[i] > 126 {
			d, _ := json.Marshal(string(s))
			return append(b, d...)
		}
	}
	b = append(b, '"')
	b = append(b, s...)
	b = append(b, '"')
	return b
}

// Rect returns the bounding box of the Object
func (o Object) Rect() (min, max [3]float64) {
	if !o.IsGeometry() {
		return
	}
	tail := o.data[len(o.data)-1]
	if tail>>2&1 == 1 {
		if tail>>1&1 == 1 {
			if len(o.data) < 49 {
				return
			}
			// 3D RECT
			min[0] = math.Float64frombits(binary.LittleEndian.Uint64(o.data))
			min[1] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[8:]))
			min[2] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[16:]))
			max[0] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[24:]))
			max[1] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[32:]))
			max[2] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[40:]))
		} else {
			if len(o.data) < 33 {
				return
			}
			// 2D RECT
			min[0] = math.Float64frombits(binary.LittleEndian.Uint64(o.data))
			min[1] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[8:]))
			max[0] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[16:]))
			max[1] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[24:]))
		}
	} else {
		if tail>>1&1 == 1 {
			if len(o.data) < 25 {
				return
			}
			// 3D POINT
			min[0] = math.Float64frombits(binary.LittleEndian.Uint64(o.data))
			min[1] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[8:]))
			min[2] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[16:]))
			max[0] = min[0]
			max[1] = min[1]
			max[2] = min[2]
		} else {
			if len(o.data) < 17 {
				return
			}
			// 2D POINT
			min[0] = math.Float64frombits(binary.LittleEndian.Uint64(o.data))
			min[1] = math.Float64frombits(binary.LittleEndian.Uint64(o.data[8:]))
			max[0] = min[0]
			max[1] = min[1]
		}
	}
	return
}

// Point returns a point that represents the center position of the Object
func (o Object) Position() Position {
	min, max := o.Rect()
	return Position{
		(max[0] + min[0]) / 2,
		(max[1] + min[1]) / 2,
		(max[2] + min[2]) / 2,
	}
}

// BBox returns the bounding box of the object.
func (o Object) BBox() BBox {
	min, max := o.Rect()
	return BBox{
		Min: Position{min[0], min[1], min[2]},
		Max: Position{max[0], max[1], max[2]},
	}
}

// Binary returns the raw geobin bytes.
func (o Object) Binary() []byte {
	return o.data[:len(o.data):len(o.data)]
}

// SetExData creates a copy of the object with the ExData component set to
// the specified data. The original object is not altered.
func (o Object) SetExData(data []byte) Object {
	c := o.parseComponents()
	c.exdata = data
	return c.reconstructObject()
}

// ExData returns the ExData component of the object.
func (o Object) ExData() []byte {
	c := o.parseComponents()
	return c.exdata
}

// Members returns the Members component of the object. This is a JSON
// document containing the "id" and "properties" members.
// returns nil if no members are defined.
func (o Object) Members() []byte {
	if len(o.data) == 0 {
		return nil
	}
	tail := o.data[len(o.data)-1]
	if tail&1 == 0 {
		return nil
	}
	if (tail>>3)&1 == 0 {
		return nil
	}
	var bboxSize int
	if tail>>1&1 == 1 {
		if tail>>2&1 == 1 {
			bboxSize = 48
		} else {
			bboxSize = 24
		}
	} else {
		if tail>>2&1 == 1 {
			bboxSize = 32
		} else {
			bboxSize = 16
		}
	}
	// complex, let's pull the geom data
	geomData := o.data[bboxSize:]
	if geomData[0]&1 == 1 {
		sz := int(binary.LittleEndian.Uint32(geomData[1:]))
		return geomData[5 : 5+sz : 5+sz]
	}
	return nil
}

// Dims returns the number of dimensions for the geometry object.
// The result will be 0, 2, or 3.
func (o Object) Dims() int {
	if len(o.data) == 0 {
		return 0
	}
	tail := o.data[len(o.data)-1]
	if tail&1 == 0 {
		return 0
	}
	if tail>>1&1 == 1 {
		return 3
	}
	return 2
}

type components struct {
	tail   byte
	bbox   []byte
	exdata []byte
	data   []byte
}

func (o Object) parseComponents() (c components) {
	if len(o.data) == 0 {
		return
	}
	var bboxSize int
	c.tail = o.data[len(o.data)-1]
	if c.tail>>0&1 == 1 {
		// geom
		if c.tail>>1&1 == 1 {
			if c.tail>>2&1 == 1 {
				// 3D rect
				bboxSize = 48
			} else {
				// 3D point
				bboxSize = 24
			}
		} else {
			if c.tail>>2&1 == 1 {
				// 2D rect
				bboxSize = 32
			} else {
				// 2D point
				bboxSize = 16
			}
		}
		c.bbox = o.data[:bboxSize]
	}

	if c.tail>>4&1 == 1 {
		// haseexdata
		exdataSize := int(binary.LittleEndian.Uint32(o.data[len(o.data)-5:]))
		c.exdata = o.data[len(o.data)-5-exdataSize : len(o.data)-5]
		c.data = o.data[bboxSize : len(o.data)-5-exdataSize]
	} else {
		c.data = o.data[bboxSize : len(o.data)-1]
	}
	return c
}

func (c *components) reconstructObject() Object {
	var exdataSizeSize int
	if len(c.exdata) > 0 {
		exdataSizeSize = 4
	}
	raw := make([]byte, len(c.bbox)+len(c.data)+len(c.exdata)+exdataSizeSize+1)
	copy(raw, c.bbox)
	if exdataSizeSize > 0 {
		raw[len(raw)-1] = c.tail | byte(1<<4)
		binary.LittleEndian.PutUint32(raw[len(raw)-5:], uint32(len(c.exdata)))
		copy(raw[len(raw)-5-len(c.exdata):], c.exdata)
	} else {
		raw[len(raw)-1] = c.tail & ^byte(1<<4)
	}
	copy(raw[len(c.bbox):], c.data)
	return Object{raw}
}

func makeSixAny(tail byte, count int, any [6]float64) Object {
	data := make([]byte, count*8+1)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint64(data[i*8:], math.Float64bits(any[i]))
	}
	data[count*8] = tail
	return Object{data}
}

// appendGeojsonBBox appends a geojson bbox in the form of `,"bbox":[n,n,n,n]`.
// the number of dimensions is determined by the the width of the slice.
func appendGeojsonBBox(json, bbox []byte) []byte {
	if len(bbox) > 0 {
		json = append(json, `,"bbox":[`...)
		for i := 0; i < len(bbox); i += 8 {
			if i > 0 {
				json = append(json, ',')
			}
			v := math.Float64frombits(binary.LittleEndian.Uint64(bbox[i:]))
			json = strconv.AppendFloat(json, v, 'f', -1, 64)
		}
		json = append(json, ']')
	}
	return json
}
func appendGeojsonGeometries(json, data []byte) []byte {
	json = append(json, '[')
	n := int(binary.LittleEndian.Uint32(data))
	data = data[4:]
	for i := 0; i < n; i++ {
		if i > 0 {
			json = append(json, ',')
		}
		sz := int(binary.LittleEndian.Uint32(data))
		o := Object{data[4 : 4+sz]}
		data = data[4+sz:]
		json = appendGeojsonBytes(json, o)
	}
	json = append(json, ']')
	return json
}
func appendGeojsonCoordinates(json, data []byte, depth, dims int) ([]byte, []byte) {
	json = append(json, '[')
	if depth == 0 {
		for i := 0; i < dims; i++ {
			if i > 0 {
				json = append(json, ',')
			}
			v := math.Float64frombits(binary.LittleEndian.Uint64(data))
			data = data[8:]
			json = strconv.AppendFloat(json, v, 'f', -1, 64)
		}
	} else {
		n := int(binary.LittleEndian.Uint32(data))
		data = data[4:]
		for i := 0; i < n; i++ {
			if i > 0 {
				json = append(json, ',')
			}
			json, data = appendGeojsonCoordinates(json, data, depth-1, dims)
		}
	}
	json = append(json, ']')
	return json, data
}

func appendGeojsonComplexBytes(json []byte, o Object) []byte {
	json = append(json, '{')
	c := o.parseComponents()
	var dims int
	if c.tail>>1&1 == 1 {
		dims = 3
	} else {
		dims = 2
	}
	data := c.data
	if len(data) != 0 {
		hasMembers := data[0]&1 == 1
		exportedBBox := data[0]>>1&1 == 1
		typ := GeometryType(data[0] >> 4)
		data = data[1:]
		var members []byte
		if hasMembers {
			sz := int(binary.LittleEndian.Uint32(data))
			members = data[4 : 4+sz]
			data = data[4+sz:]
		}
		var depth int
		switch typ {
		default:
			json = append(json, `"type":"Unknown"`...)
		case Point:
			json, depth = append(json, `"type":"Point"`...), 0
		case MultiPoint:
			json, depth = append(json, `"type":"MultiPoint"`...), 1
		case LineString:
			json, depth = append(json, `"type":"LineString"`...), 1
		case MultiLineString:
			json, depth = append(json, `"type":"MultiLineString"`...), 2
		case Polygon:
			json, depth = append(json, `"type":"Polygon"`...), 2
		case MultiPolygon:
			json, depth = append(json, `"type":"MultiPolygon"`...), 3
		case GeometryCollection:
			json = append(json, `"type":"GeometryCollection"`...)
		case Feature:
			json = append(json, `"type":"Feature"`...)
		case FeatureCollection:
			json = append(json, `"type":"FeatureCollection"`...)
		}
		if exportedBBox {
			json = appendGeojsonBBox(json, c.bbox)
		}
		if typ > Unknown && typ <= MultiPolygon {
			json = append(json, `,"coordinates":`...)
			json, data = appendGeojsonCoordinates(json, data, depth, dims)
		} else if typ == GeometryCollection {
			json = append(json, `,"geometries":`...)
			json = appendGeojsonGeometries(json, data)
		} else if typ == Feature {
			json = append(json, `,"geometry":`...)
			sz := int(binary.LittleEndian.Uint32(data))
			g := Object{data[4 : 4+sz : 4+sz]}
			json = appendGeojsonBytes(json, g)
		} else if typ == FeatureCollection {
			json = append(json, `,"features":`...)
			json = appendGeojsonGeometries(json, data)
		}
		if len(members) > 2 && members[0] == '{' && members[len(members)-1] == '}' {
			json = append(json, ',')
			json = append(json, members[1:len(members)-1]...)
		}
	}
	json = append(json, '}')
	return json
}

func appendGeojsonBytesPolygon(json []byte, pairs [][]float64) []byte {
	json = append(json, '[')
	for i, pair := range pairs {
		if i > 0 {
			json = append(json, ',')
		}
		json = append(json, '[')
		for j, val := range pair {
			if j > 0 {
				json = append(json, ',')
			}
			json = strconv.AppendFloat(json, val, 'f', -1, 64)
		}
		json = append(json, ']')
	}
	json = append(json, ']')
	return json
}

func (o Object) simplePairsFor2DRect() [][]float64 {
	min, max := o.Rect()
	return [][]float64{
		{min[0], min[1]}, {max[0], min[1]}, {max[0], max[1]}, {min[0], max[1]}, {min[0], min[1]},
	}
}
func (o Object) simplePairsFor3DRect() [][][]float64 {
	min, max := o.Rect()
	return [][][]float64{
		// bottom
		{{min[0], min[1], min[2]}, {max[0], min[1], min[2]}, {max[0], max[1], min[2]}, {min[0], max[1], min[2]}, {min[0], min[1], min[2]}},
		// north
		{{min[0], max[1], min[2]}, {max[0], max[1], min[2]}, {max[0], max[1], max[2]}, {min[0], max[1], max[2]}, {min[0], max[1], min[2]}},
		// south
		{{min[0], min[1], min[2]}, {max[0], min[1], min[2]}, {max[0], min[1], max[2]}, {min[0], min[1], max[2]}, {min[0], min[1], min[2]}},
		// west
		{{min[0], min[1], min[2]}, {min[0], max[1], min[2]}, {min[0], max[1], max[2]}, {min[0], min[1], max[2]}, {min[0], min[1], min[2]}},
		// east
		{{max[0], min[1], min[2]}, {max[0], max[1], min[2]}, {max[0], max[1], max[2]}, {max[0], min[1], max[2]}, {max[0], min[1], min[2]}},
		//top
		{{min[0], min[1], max[2]}, {max[0], min[1], max[2]}, {max[0], max[1], max[2]}, {min[0], max[1], max[2]}, {min[0], min[1], max[2]}},
	}
}
func appendGeojsonBytes(json []byte, o Object) []byte {
	if o.data[len(o.data)-1]>>3&1 == 1 {
		return appendGeojsonComplexBytes(json, o)
	}
	switch o.data[len(o.data)-1] & 15 {
	default:
		// invalid
		return append(json, `{"type":"Unknown"}`...)
	case 1:
		// simple 2D point
		p := o.Position()
		json := append(json, `{"type":"Point","coordinates":[`...)
		json = strconv.AppendFloat(json, p.X, 'f', -1, 64)
		json = append(json, ',')
		json = strconv.AppendFloat(json, p.Y, 'f', -1, 64)
		return append(json, ']', '}')
	case 3:
		// simple 3D point
		p := o.Position()
		json := append(json, `{"type":"Point","coordinates":[`...)
		json = strconv.AppendFloat(json, p.X, 'f', -1, 64)
		json = append(json, ',')
		json = strconv.AppendFloat(json, p.Y, 'f', -1, 64)
		json = append(json, ',')
		json = strconv.AppendFloat(json, p.Z, 'f', -1, 64)
		return append(json, ']', '}')
	case 5:
		// simple 2D rect
		json := append(json, `{"type":"Polygon","coordinates":[`...)
		json = appendGeojsonBytesPolygon(json, o.simplePairsFor2DRect())
		return append(json, ']', '}')
	case 7:
		// simple 3D rect
		pairs := o.simplePairsFor3DRect()
		json := append(json, `{"type":"MultiPolygon","coordinates":[`...)
		for i := 0; i < len(pairs); i++ {
			if i > 0 {
				json = append(json, ',')
			}
			json = append(json, '[')
			json = appendGeojsonBytesPolygon(json, pairs[i])
			json = append(json, ']')
		}
		return append(json, ']', '}')
	}
}

var baseMin = [3]float64{math.Inf(+1), math.Inf(+1), math.Inf(+1)}
var baseMax = [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}

func valsFromCoords0(coords gjson.Result) (vals [3]float64, dims int) {
	coords.ForEach(func(_, val gjson.Result) bool {
		vals[dims] = val.Float()
		dims++
		return dims < 3
	})
	if dims < 2 {
		dims = 2
	}
	return vals, dims
}

func valsFromCoords1(coords gjson.Result, min, max [3]float64) (vals [][3]float64, dims int, minOut, maxOut [3]float64) {
	coords.ForEach(func(_, val gjson.Result) bool {
		var tvals [3]float64
		tvals, dims = valsFromCoords0(val)
		for i := 0; i < dims; i++ {
			if tvals[i] < min[i] {
				min[i] = tvals[i]
			}
			if tvals[i] > max[i] {
				max[i] = tvals[i]
			}
		}
		vals = append(vals, tvals)
		return true
	})
	return vals, dims, min, max
}

func valsFromCoords2(coords gjson.Result, min, max [3]float64) (vals [][][3]float64, dims int, minOut, maxOut [3]float64) {
	coords.ForEach(func(_, val gjson.Result) bool {
		var tvals [][3]float64
		tvals, dims, min, max = valsFromCoords1(val, min, max)
		vals = append(vals, tvals)
		return true
	})
	return vals, dims, min, max
}

func valsFromCoords3(coords gjson.Result, min, max [3]float64) (vals [][][][3]float64, dims int, minOut, maxOut [3]float64) {
	coords.ForEach(func(_, val gjson.Result) bool {
		var tvals [][][3]float64
		tvals, dims, min, max = valsFromCoords2(val, min, max)
		vals = append(vals, tvals)
		return true
	})
	return vals, dims, min, max
}
func appendGeomData1(data []byte, vals [][3]float64, dims int) []byte {
	data = append(data, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(data[len(data)-4:], uint32(len(vals)))
	for i := 0; i < len(vals); i++ {
		for j := 0; j < dims; j++ {
			data = append(data, 0, 0, 0, 0, 0, 0, 0, 0)
			binary.LittleEndian.PutUint64(data[len(data)-8:], math.Float64bits(vals[i][j]))
		}
	}
	return data
}
func appendGeomData2(data []byte, vals [][][3]float64, dims int) []byte {
	data = append(data, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(data[len(data)-4:], uint32(len(vals)))
	for i := 0; i < len(vals); i++ {
		data = appendGeomData1(data, vals[i], dims)
	}
	return data
}
func appendGeomData3(data []byte, vals [][][][3]float64, dims int) []byte {
	data = append(data, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(data[len(data)-4:], uint32(len(vals)))
	for i := 0; i < len(vals); i++ {
		data = appendGeomData2(data, vals[i], dims)
	}
	return data
}
func tailFromBBoxJSONOrMakeIfNeeded(bbox gjson.Result, min, max [3]float64, dims int) (tail byte, bboxData []byte, exportBBox bool) {
	tail, bboxData = tailFromBBoxJSON(bbox)
	if tail == 0 {
		// we don't have a bbox, make one now
		if dims == 2 {
			bboxData = make([]byte, 32)
		} else {
			bboxData = make([]byte, 48)
		}
		for i := 0; i < dims; i++ {
			binary.LittleEndian.PutUint64(bboxData[i*8:], math.Float64bits(min[i]))
			binary.LittleEndian.PutUint64(bboxData[dims*8+i*8:], math.Float64bits(max[i]))
		}
		if dims == 2 {
			tail = 13
		} else if dims == 3 {
			tail = 15
		} else {
			panic("invalid dims")
		}
	} else {
		// user defined bbox is available
		exportBBox = true
	}
	return tail, bboxData, exportBBox
}
func tailFromBBoxJSON(bbox gjson.Result) (tail byte, data []byte) {
	if !bbox.Exists() {
		return 0, nil
	}
	var count int
	var vals [8]float64
	bbox.ForEach(func(_, val gjson.Result) bool {
		if count == len(vals) {
			count = 0 // invalid
			return false
		}
		vals[count] = val.Float()
		count++
		return true
	})
	if count < 4 || count%2 != 0 {
		// ignore invalid bboxes
		return 0, nil
	}
	if count > 6 {
		// make sure to convert bboxes greater than 3 dims to 3 dims.
		copy(vals[3:], vals[count/2:count/2+3])
		count = 6
	}
	// now we have a bbox that is 4 or 6 wide
	data = make([]byte, count*8)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint64(data[i*8:], math.Float64bits(vals[i]))
	}
	if count == 4 {
		return 13, data
	}
	return 15, data
}

func level1FromJSON(typ GeometryType, bbox, coords gjson.Result) Object {
	vals, dims, min, max := valsFromCoords1(coords, baseMin, baseMax)
	if dims < 2 {
		dims = 2
	}
	tail, raw, exportBBox := tailFromBBoxJSONOrMakeIfNeeded(bbox, min, max, dims)
	dims = len(raw) / 16 // clip to bbox dims
	if exportBBox {
		raw = append(raw, (byte(typ)<<4)|2)
	} else {
		raw = append(raw, byte(typ)<<4)
	}
	raw = appendGeomData1(raw, vals, dims)
	raw = append(raw, tail)
	return Object{raw}
}

func level2FromJSON(typ GeometryType, bbox, coords gjson.Result) Object {
	vals, dims, min, max := valsFromCoords2(coords, baseMin, baseMax)
	if dims < 2 {
		dims = 2
	}
	tail, raw, exportBBox := tailFromBBoxJSONOrMakeIfNeeded(bbox, min, max, dims)
	dims = len(raw) / 16 // clip to bbox dims
	// check if it's a simple 2D rectangle
	if !exportBBox && typ == Polygon && dims == 2 && len(vals) == 1 {
		if polyRectIsNormal(vals[0], 0, 1, 2) {
			// simple 2D rectangle
			return Make2DRect(min[0], min[1], max[0], max[1])
		}
	}
	if exportBBox {
		raw = append(raw, (byte(typ)<<4)|2)
	} else {
		raw = append(raw, byte(typ)<<4)
	}
	raw = appendGeomData2(raw, vals, dims)
	raw = append(raw, tail)
	return Object{raw}
}

func polyRectIsNormal(points [][3]float64, x, y, z int) bool {
	if len(points) != 5 {
		return false
	}
	zz := points[0][z]
	for i := 0; i < 5; i++ {
		if points[i][z] != zz {
			return false
		}
	}
	if points[0][x] != points[3][x] {
		return false
	}
	if points[0][x] != points[4][x] {
		return false
	}
	if points[1][x] != points[2][x] {
		return false
	}
	if points[0][y] != points[1][y] {
		return false
	}
	if points[0][y] != points[4][y] {
		return false
	}
	if points[2][y] != points[3][y] {
		return false
	}
	return true
}
func level3FromJSON(typ GeometryType, bbox, coords gjson.Result) Object {
	vals, dims, min, max := valsFromCoords3(coords, baseMin, baseMax)
	if dims < 2 {
		dims = 2
	}
	tail, raw, exportBBox := tailFromBBoxJSONOrMakeIfNeeded(bbox, min, max, dims)
	dims = len(raw) / 16 // clip to bbox dims
	if !exportBBox && typ == MultiPolygon && dims == 3 && len(vals) == 6 {
		simple := true
		orders := [6][3]int{
			// bottom
			{0, 1, 2},
			// north
			{0, 2, 1},
			// south
			{0, 2, 1},
			// west
			{1, 2, 0},
			// east
			{1, 2, 0},
			// top
			{0, 1, 2},
		}
		for i := 0; i < len(vals); i++ {
			var ok bool
			if len(vals[i]) == 1 {
				ok = polyRectIsNormal(vals[i][0], orders[i][0], orders[i][1], orders[i][2])
			}
			if !ok {
				simple = false
				break
			}
		}
		if simple {
			return Make3DRect(min[0], min[1], min[2], max[0], max[1], max[2])
		}
	}
	if exportBBox {
		raw = append(raw, (byte(typ)<<4)|2)
	} else {
		raw = append(raw, byte(typ)<<4)
	}
	raw = appendGeomData3(raw, vals, dims)
	raw = append(raw, tail)
	return Object{raw}
}
func pointFromJSON(bbox, coords gjson.Result) Object {
	typ := Point
	vals, dims := valsFromCoords0(coords)
	if dims < 2 {
		dims = 2
	}
	tail, bboxData := tailFromBBoxJSON(bbox)
	if tail == 0 {
		// use simple a object
		if dims == 2 {
			return Make2DPoint(vals[0], vals[1])
		}
		return Make3DPoint(vals[0], vals[1], vals[2])
	}
	// clip the dims to the bbox
	dims = len(bboxData) / 16
	var data []byte
	if dims == 3 {
		// [RAW] = [BBOX][HEAD][X][Y][Z][TAIL] = 50 bytes
		data = make([]byte, 74)
		copy(data, bboxData)
		data[48] = (byte(typ) << 4) | 2 // export bbox
		binary.LittleEndian.PutUint64(data[49:], math.Float64bits(vals[0]))
		binary.LittleEndian.PutUint64(data[57:], math.Float64bits(vals[1]))
		binary.LittleEndian.PutUint64(data[65:], math.Float64bits(vals[2]))
		data[73] = tail
	} else {
		// [RAW] = [BBOX][HEAD][X][Y][TAIL] = 34 bytes
		data = make([]byte, 50)
		copy(data, bboxData)
		data[32] = (byte(typ) << 4) | 2 // export bbox
		binary.LittleEndian.PutUint64(data[33:], math.Float64bits(vals[0]))
		binary.LittleEndian.PutUint64(data[41:], math.Float64bits(vals[1]))
		data[49] = tail
	}
	return Object{data}
}

func collectionFromJSON(typ GeometryType, bbox, geoms gjson.Result) Object {
	var dims int
	min, max := baseMin, baseMax
	var vals []Object
	var invalid bool
	geoms.ForEach(func(_, val gjson.Result) bool {
		g := objectFromJSON(val.Raw)
		if !g.IsGeometry() {
			invalid = true
			return false
		}
		vals = append(vals, g)
		gdims := g.Dims()
		if gdims > dims {
			dims = gdims
		}
		gmin, gmax := g.Rect()
		for i := 0; i < gdims; i++ {
			if gmin[i] < min[i] {
				min[i] = gmin[i]
			}
			if gmax[i] > max[i] {
				max[i] = gmax[i]
			}
		}
		return true
	})
	if invalid {
		return Object{}
	}
	if dims < 2 {
		dims = 2
	}
	tail, raw, exportBBox := tailFromBBoxJSONOrMakeIfNeeded(bbox, min, max, dims)
	dims = len(raw) / 16 // clip to bbox dims
	if exportBBox {
		raw = append(raw, (byte(typ)<<4)|2)
	} else {
		raw = append(raw, byte(typ)<<4)
	}

	raw = append(raw, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(raw[len(raw)-4:], uint32(len(vals)))
	for i := 0; i < len(vals); i++ {
		raw = append(raw, 0, 0, 0, 0)
		data := vals[i].data
		binary.LittleEndian.PutUint32(raw[len(raw)-4:], uint32(len(data)))
		raw = append(raw, data...)
	}
	raw = append(raw, tail)
	return Object{raw}
}
func featureFromJSON(bbox, geom, id, props gjson.Result) Object {
	typ := byte(Feature)
	g := objectFromJSON(geom.Raw)
	if !g.IsGeometry() {
		return Object{}
	}
	var exportBBox bool
	tail, bboxData := tailFromBBoxJSON(bbox)
	if tail == 0 {
		// use the bbox from the geom
		gtail := g.data[len(g.data)-1]
		if gtail>>1&1 == 1 {
			if gtail>>2&1 == 1 {
				bboxData = g.data[:48:48]
				tail = 15
			} else {
				bboxData = g.data[:24:24]
				tail = 11
			}
		} else {
			if gtail>>2&1 == 1 {
				bboxData = g.data[:32:32]
				tail = 13
			} else {
				bboxData = g.data[:16:16]
				tail = 9
			}
		}
	} else {
		exportBBox = true
	}
	// create the members json block
	var members []byte
	if id.Exists() || props.Exists() {
		members = append(members, '{')
		if id.Exists() {
			members = append(members, `"id":`...)
			members = append(members, id.Raw...)
		}
		if props.Exists() {
			if len(members) > 1 {
				members = append(members, ',')
			}
			members = append(members, `"properties":`...)
			members = append(members, props.Raw...)
		}
		members = append(members, '}')
		members = pretty.UglyInPlace(members)
	}
	// build the object
	raw := bboxData
	if exportBBox {
		if len(members) > 0 {
			raw = append(raw, (typ<<4)|3)
		} else {
			raw = append(raw, (typ<<4)|2)
		}
	} else {
		if len(members) > 0 {
			raw = append(raw, (typ<<4)|1)
		} else {
			raw = append(raw, typ<<4)
		}
	}
	if len(members) > 0 {
		raw = append(raw, 0, 0, 0, 0)
		binary.LittleEndian.PutUint32(raw[len(raw)-4:], uint32(len(members)))
		raw = append(raw, members...)
	}
	raw = append(raw, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(raw[len(raw)-4:], uint32(len(g.data)))
	raw = append(raw, g.data...)
	raw = append(raw, tail)
	return Object{raw}
}

func objectFromJSON(json string) Object {
	switch gjson.Get(json, "type").String() {
	default:
		return Object{}
	case "Point":
		return pointFromJSON(gjson.Get(json, "bbox"), gjson.Get(json, "coordinates"))
	case "MultiPoint":
		return level1FromJSON(MultiPoint, gjson.Get(json, "bbox"), gjson.Get(json, "coordinates"))
	case "LineString":
		return level1FromJSON(LineString, gjson.Get(json, "bbox"), gjson.Get(json, "coordinates"))
	case "MultiLineString":
		return level2FromJSON(MultiLineString, gjson.Get(json, "bbox"), gjson.Get(json, "coordinates"))
	case "Polygon":
		return level2FromJSON(Polygon, gjson.Get(json, "bbox"), gjson.Get(json, "coordinates"))
	case "MultiPolygon":
		return level3FromJSON(MultiPolygon, gjson.Get(json, "bbox"), gjson.Get(json, "coordinates"))
	case "GeometryCollection":
		return collectionFromJSON(GeometryCollection, gjson.Get(json, "bbox"), gjson.Get(json, "geometries"))
	case "Feature":
		var id, props, geom, bbox gjson.Result
		gjson.Parse(json).ForEach(func(key, val gjson.Result) bool {
			switch key.String() {
			case "id":
				id = val
			case "properties":
				props = val
			case "geometry":
				geom = val
			case "bbox":
				bbox = val
			}
			return true
		})
		return featureFromJSON(bbox, geom, id, props)
	case "FeatureCollection":
		return collectionFromJSON(FeatureCollection, gjson.Get(json, "bbox"), gjson.Get(json, "features"))
	}
}

// GeometryType represents a geojson geometry type
type GeometryType byte

const (
	Unknown GeometryType = iota
	Point
	MultiPoint
	LineString
	MultiLineString
	Polygon
	MultiPolygon
	GeometryCollection
	Feature
	FeatureCollection
)

func (t GeometryType) String() string {
	switch t {
	default:
		return "Unknown"
	case Point:
		return "Point"
	case MultiPoint:
		return "MultiPoint"
	case LineString:
		return "LineString"
	case MultiLineString:
		return "MultiLineString"
	case Polygon:
		return "Polygon"
	case MultiPolygon:
		return "MultiPolygon"
	case GeometryCollection:
		return "GeometryCollection"
	case Feature:
		return "Feature"
	case FeatureCollection:
		return "FeatureCollection"
	}
}

// GeometryType returns the geometry type for the object.
func (o Object) GeometryType() GeometryType {
	if len(o.data) == 0 {
		return Unknown
	}
	tail := o.data[len(o.data)-1]
	if tail&1 == 0 {
		return Unknown
	}
	var bboxSize int
	if tail>>1&1 == 1 {
		if tail>>2&1 == 1 {
			// 3D rect
			bboxSize = 48
		} else {
			// 3D point
			bboxSize = 24
		}
	} else {
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
		case 48:
			// 3D rect -> MultiPolygon
			return MultiPolygon
		case 32:
			// 2D rect -> Polygon
			return Polygon
		case 24, 16:
			return Point
		}
	}
	// complex
	return GeometryType(o.data[bboxSize] >> 4)
}

func (o Object) polySimplePairsFor2DRect() []Position {
	min, max := o.Rect()
	return []Position{
		{min[0], min[1], 0}, {max[0], min[1], 0}, {max[0], max[1], 0},
		{min[0], max[1], 0}, {min[0], min[1], 0},
	}
}

func (o Object) polySimplePairsFor3DRect() [][]Position {
	min, max := o.Rect()
	return [][]Position{
		// bottom
		{{min[0], min[1], min[2]}, {max[0], min[1], min[2]}, {max[0], max[1], min[2]}, {min[0], max[1], min[2]}, {min[0], min[1], min[2]}},
		// north
		{{min[0], max[1], min[2]}, {max[0], max[1], min[2]}, {max[0], max[1], max[2]}, {min[0], max[1], max[2]}, {min[0], max[1], min[2]}},
		// south
		{{min[0], min[1], min[2]}, {max[0], min[1], min[2]}, {max[0], min[1], max[2]}, {min[0], min[1], max[2]}, {min[0], min[1], min[2]}},
		// west
		{{min[0], min[1], min[2]}, {min[0], max[1], min[2]}, {min[0], max[1], max[2]}, {min[0], min[1], max[2]}, {min[0], min[1], min[2]}},
		// east
		{{max[0], min[1], min[2]}, {max[0], max[1], min[2]}, {max[0], max[1], max[2]}, {max[0], min[1], max[2]}, {max[0], min[1], min[2]}},
		//top
		{{min[0], min[1], max[2]}, {max[0], min[1], max[2]}, {max[0], max[1], max[2]}, {min[0], max[1], max[2]}, {min[0], min[1], max[2]}},
	}
}

// readFloat64 reads a float64 from a data.
func readFloat64(data []byte) (float64, []byte) {
	f := math.Float64frombits(binary.LittleEndian.Uint64(data))
	return f, data[8:]
}

// readUint32 reads a uint32 from data and returns as an int.
func readUint32(data []byte) (int, []byte) {
	return int(binary.LittleEndian.Uint32(data)), data[4:]
}

// readPosition reads a point from a data and returns the new point.
func readPosition(data []byte, dims int) (Position, []byte) {
	var p Position
	p.X, data = readFloat64(data)
	p.Y, data = readFloat64(data)
	if dims == 3 {
		p.Z, data = readFloat64(data)
	}
	return p, data
}

/*
// Geometry returns the underlying geometry points or collection. The geom is
// one of the following for the geomType:
// Unknown -> nil;
// Point -> Position;
// MultiPoint, LineString -> []Position;
// MultiLineString, Polygon -> [][]Position;
// MultiPolygon -> [][][]Position;
// GeometryCollection, FeatureCollection -> []Object;
// Dims is zero for Unknown, GeometryCollection, and FeatureCollection;
// Otherwise it's 2 or 3.
func (o Object) Geometry() (geom interface{}, dims int, geomType GeometryType) {
	if len(o.data) == 0 {
		// empty geometry
		return
	}
	tail := o.data[len(o.data)-1]
	if tail&1 == 0 {
		// object is a string
		return
	}
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
		case 48:
			// simple 3d rect
			geomType = MultiPolygon
			geom = [][][]Position{o.polySimplePairsFor3DRect()}
		case 32:
			// simple 2d rect
			geomType = Polygon
			geom = [][]Position{o.polySimplePairsFor2DRect()}
		case 24:
			// simple 3d point
			geomType = Point
			geom, _ = polyReadPosition(o.data, 3)
		case 16:
			// simple 2d point
			geomType = Point
			geom, _ = polyReadPosition(o.data, 2)

		}
		return
	}
	// complex, let's pull the geom data
	var exsz int
	if tail>>4&1 == 1 {
		// has exdata, skip over
		exsz = int(binary.LittleEndian.Uint32(o.data[len(o.data)-5:]))
	}
	geomData := o.data[bboxSize+exsz:]
	geomHead := geomData[0]
	hasMembers := geomHead&1 == 1
	geomType = GeometryType(geomHead >> 4)
	geomData = geomData[1:]
	if hasMembers {
		sz := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4+sz:]
	}
	switch geomType {
	case Point:
		geom, _ = polyReadPosition(geomData, dims)
	case MultiPoint, LineString:
		n := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4:]
		points := make([]Position, n)
		for i := 0; i < n; i++ {
			points[i], geomData = polyReadPosition(geomData, dims)
		}
		geom = points
	case MultiLineString, Polygon:
		n := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4:]
		points := make([][]Position, n)
		for i := 0; i < n; i++ {
			nn := int(binary.LittleEndian.Uint32(geomData))
			geomData = geomData[4:]
			points[i] = make([]Position, nn)
			for j := 0; j < nn; j++ {
				points[i][j], geomData = polyReadPosition(geomData, dims)
			}
		}
		geom = points
	case MultiPolygon:
		n := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4:]
		points := make([][][]Position, n)
		for i := 0; i < n; i++ {
			nn := int(binary.LittleEndian.Uint32(geomData))
			geomData = geomData[4:]
			points[i] = make([][]Position, nn)
			for j := 0; j < nn; j++ {
				nnn := int(binary.LittleEndian.Uint32(geomData))
				geomData = geomData[4:]
				points[i][j] = make([]Position, nnn)
				for k := 0; k < nnn; k++ {
					points[i][j][k], geomData = polyReadPosition(geomData, dims)
				}
			}
		}
		geom = points
	case GeometryCollection, FeatureCollection:
		dims = 0
		n := int(binary.LittleEndian.Uint32(geomData))
		geomData = geomData[4:]
		objs := make([]Object, n)
		for i := 0; i < n; i++ {
			sz := int(binary.LittleEndian.Uint32(geomData))
			o := Object{geomData[4 : 4+sz : 4+sz]}
			geomData = geomData[4+sz:]
			objs[i] = o
		}
		geom = objs
	case Feature:
		sz := int(binary.LittleEndian.Uint32(geomData))
		return Object{geomData[4 : 4+sz : 4+sz]}.Geometry()
	}
	return
}
*/

// PositionCount returns the total number of points in the geometry.
func (o Object) PositionCount() int {
	return o.Geometry().PositionCount()
}

type Geometry struct {
	Data   []byte
	Dims   int
	Type   GeometryType
	Simple bool
}

func (o Object) Geometry() Geometry {
	var geom Geometry
	if len(o.data) == 0 {
		return geom // not a geometry
	}
	tail := o.data[len(o.data)-1]
	if tail&1 == 0 {
		return geom // not a geometry
	}
	var bboxSize int
	if tail>>1&1 == 1 {
		geom.Dims = 3
		if tail>>2&1 == 1 {
			bboxSize = 48 // 3D rect
		} else {
			bboxSize = 24 // 3D point
		}
	} else {
		geom.Dims = 2
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
		geom.Simple = true
		geom.Data = o.data
		switch bboxSize {
		case 48:
			geom.Type = MultiPolygon
		case 32:
			geom.Type = Polygon
		case 24:
			geom.Type = Point
		case 16:
			geom.Type = Point
		}
		return geom
	}
	// complex, let's pull the geom data
	geom.Data = o.data[bboxSize:]
	geom.Type = GeometryType(geom.Data[0] >> 4)
	if geom.Data[0]&1 == 1 {
		// has members, skip over
		sz := int(binary.LittleEndian.Uint32(geom.Data[1:]))
		geom.Data = geom.Data[5+sz:]
	} else {
		geom.Data = geom.Data[1:]
	}
	return geom
}
func (g Geometry) PositionCount() int {
	if g.Type == Point {
		return 1
	}
	if g.Simple {
		return 2
	}
	var count int
	switch g.Type {
	case MultiPoint, LineString:
		count, _ = readUint32(g.Data)
	case MultiLineString, Polygon:
		n, data := readUint32(g.Data)
		for i := 0; i < n; i++ {
			var nn int
			nn, data = readUint32(data)
			count += nn
			data = data[nn*8*g.Dims:]
		}
	case MultiPolygon:
		n, data := readUint32(g.Data)
		for i := 0; i < n; i++ {
			var nn int
			nn, data = readUint32(data)
			for i := 0; i < nn; i++ {
				var nnn int
				nnn, data = readUint32(data)
				count += nnn
				data = data[nnn*8*g.Dims:]
			}
		}
	case GeometryCollection, FeatureCollection:
		n, data := readUint32(g.Data)
		for i := 0; i < n; i++ {
			var sz int
			sz, data = readUint32(data)
			o := Object{data[:sz]}
			count += o.Geometry().PositionCount()
			data = data[sz:]
		}
	case Feature:
		sz, data := readUint32(g.Data)
		o := Object{data[:sz]}
		count = o.Geometry().PositionCount()
	}
	return count
}
