go-kinet - Color Kinetics library
========================================

A golang library for interfacing with the 
[Color Kinetics](http://www.colorkinetics.com/) LED lights.

Features
--------

- Controller and fixture discovery
- Individual fixture control
- Controller renaming

Examples
--------

The following examples are included:

**discover**: discover and list the properties of available controllers and fixtures

**rainbow**: discover all local fixtures and stream a rainbow fade to all of them

See other
---------

[chromaticity](): A 
[Philips Hue compatible REST API](http://www.developers.meethue.com/philips-hue-api) 
to control RGB(+) LED lights, includes go-kinet as a backend
[kinet](https://github.com/vishnubob/kinet): A python library for controlling
Color Kinetics lights (and a large help in developing this library!)
