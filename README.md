# Geobin
<a href="https://godoc.org/github.com/tidwall/geobin"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>

The Geobin Object represents tightly packed geometry that is compatible with [GeoJSON RFC 7946](https://tools.ietf.org/html/rfc7946).

[Specification](SPEC.md)

The purpose of this package is to provide a new binary format for Tile38 geometries.
Tile38 currently uses standard Go structs that contain nested pointer, slices, arrays, etc. 
This new format is stored as a contiguous byte stream that can be packed inside a [Pair object](https://github.com/tidwall/pair). It also precalcs the bbox for fast spatial indexing.

**This project is a (sweet) work in progress. The API will likely change between now and Tile38 v2.0 release.**

#### Notes

- Objects take up no more than one allocation.
- 2D Point size is 17 bytes
- 3D Point size is 25 bytes
- Objects have precalculated bboxes
- Polygon detection formulas (Intersects, Within, etc) are currently bridging
  the Tile38 GeoJSON package. Hopefully this will be native by Tile38 2.0 launch.


## Contact

Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

Geobin source code is available under the MIT [License](/LICENSE).
