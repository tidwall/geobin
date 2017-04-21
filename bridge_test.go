package geobin

import (
	"fmt"
	"testing"
)

func P(x, y float64) Position {
	return Position{x, y, 0}
}

func P3(x, y, z float64) Position {
	return Position{x, y, z}
}
func tPoint(x, y float64) Object {
	return ParseJSON(fmt.Sprintf(`{"type":"Point","coordinates":[%f,%f]}`, x, y))
}

const testPolyHoles = `
{"type":"Polygon","coordinates":[
[[0,0],[0,6],[12,-6],[12,0],[0,0]],
[[1,1],[1,2],[2,2],[2,1],[1,1]],
[[11,-1],[11,-3],[9,-1],[11,-1]]
]}`

func TestPointWithinBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"Point","coordinates":[10,10],"bbox":[0,0,100,100]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Point","coordinates":[10,10]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Point","coordinates":[10,10],"bbox":[-10,-10,100,100]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Point","coordinates":[-10,-10]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
}
func TestPointIntersectsBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"Point","coordinates":[10,10],"bbox":[0,0,100,100]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Point","coordinates":[10,10]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Point","coordinates":[10,10],"bbox":[-10,-10,100,100]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Point","coordinates":[10,10],"bbox":[-10,-10,0,0]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Point","coordinates":[10,10],"bbox":[-10,-10,-1,-1]}`)
	if p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Point","coordinates":[-10,-10]}`)
	if p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}

}
func TestPointWithinObject(t *testing.T) {
	p := ParseJSON(`{"type":"Point","coordinates":[10,10]}`)
	if p.Within(ParseJSON(`{"type":"Point","coordinates":[10,10],"bbox":[1,1,2,2]}`)) {
		t.Fatal("!")
	}
	if !p.Within(ParseJSON(`{"type":"Point","coordinates":[10,10],"bbox":[0,0,100,100]}`)) {
		t.Fatal("!")
	}
	poly := ParseJSON(testPolyHoles)
	ps := []Position{P(.5, 3), P(3.5, .5), P(6, 0), P(11, -1), P(11.5, -4.5)}
	expect := true
	for _, p := range ps {
		got := tPoint(p.X, p.Y).Within(poly)
		if got != expect {
			t.Fatalf("%v within = %t, expect %t", p, got, expect)
		}
	}
	ps = []Position{P(-2, 0), P(0, -2), P(1.5, 1.5), P(8, 1), P(10.5, -1.5), P(14, -1), P(8, -3)}
	expect = false
	for _, p := range ps {
		got := tPoint(p.X, p.Y).Within(poly)
		if got != expect {
			t.Fatalf("%v within = %t, expect %t", p, got, expect)
		}
	}
}
func TestMultiPointWithinBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[20,20]],"bbox":[0,0,100,100]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[20,20]]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[20,20]],"bbox":[-10,-10,100,100]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[-10,-10],[-20,-20]]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
}
func TestMultiPointIntersectsBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[20,20]],"bbox":[0,0,100,100]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[20,20]]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[20,20]],"bbox":[-10,-10,100,100]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[20,20]],"bbox":[-10,-10,0,0]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[20,20]],"bbox":[-10,-10,-1,-1]}`)
	if p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[-10,-10],[-20,-20]]}`)
	if p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiPoint","coordinates":[[10,10],[-20,-20]]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}

}
func TestLineStringWithinBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"LineString","coordinates":[[10,10],[20,20]],"bbox":[0,0,100,100]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"LineString","coordinates":[[10,10],[20,20]]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"LineString","coordinates":[[10,10],[20,20]],"bbox":[-10,-10,100,100]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"LineString","coordinates":[[-10,-10],[-20,-20]]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
}
func TestLineStringIntersectsBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"LineString","coordinates":[[10,10],[20,20]],"bbox":[0,0,100,100]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"LineString","coordinates":[[-1,3],[3,-1]]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"LineString","coordinates":[[-1,1],[1,-1]]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"LineString","coordinates":[[-2,1],[1,-1]]}`)
	if p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
}
func TestMultiLineStringWithinBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"MultiLineString","coordinates":[[[10,10],[20,20]]],"bbox":[0,0,100,100]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiLineString","coordinates":[[[10,10],[20,20]],[[30,30],[40,40]]]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiLineString","coordinates":[[[10,10],[20,20]]],"bbox":[-10,-10,100,100]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiLineString","coordinates":[[[-10,-10],[-20,-20]]]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
}

func TestMultiLineStringIntersectsBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"MultiLineString","coordinates":[[[10,10],[20,20]]],"bbox":[0,0,100,100]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiLineString","coordinates":[[[-1,3],[3,-1]],[[-1000,-1000],[-1020,-1020]]]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiLineString","coordinates":[[[-1,1],[1,-1]]]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"MultiLineString","coordinates":[[[-2,1],[1,-1]]]}`)
	if p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
}
func TestPolygonWithinBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"Polygon","coordinates":[[[10,10],[10,20],[20,10],[10,10]]],"bbox":[0,0,100,100]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Polygon","coordinates":[[[10,10],[10,20],[20,10],[10,10]]]}`)
	if !p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Polygon","coordinates":[[[10,10],[10,20],[20,10],[10,10]]],"bbox":[-10,-10,100,100]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Polygon","coordinates":[[[-10,-10],[10,20],[20,10],[-10,-10]]]}`)
	if p.WithinBBox(bbox) {
		t.Fatal("!")
	}
}

func TestPolygonIntersectsBBox(t *testing.T) {
	bbox := BBox{Min: Position{0, 0, 0}, Max: Position{100, 100, 0}}
	p := ParseJSON(`{"type":"Polygon","coordinates":[[[-10,-10],[10,20],[20,10],[-10,-10]]],"bbox":[0,0,100,100]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Polygon","coordinates":[[[-10,-10],[10,20],[20,10],[-10,-10]]]}`)
	if !p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
	p = ParseJSON(`{"type":"Polygon","coordinates":[[[-10,-10],[-30,-40],[-30,-90],[-10,-10]]]}`)
	if p.IntersectsBBox(bbox) {
		t.Fatal("!")
	}
}
