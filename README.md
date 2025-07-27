# Pretty Printer

This repo currently contains WIP code for pretty printing Golang structs using reflection.
The goals of this package is to provide simple interface allowing to:

* print structs in a human-readable way
* enable/disable pointer following. Standard library just prints pointers addresses and
  in some situations this is a problem (e.g. Apple PKL auto-generated structs have a lot of
  pointers) and user must write the printing code himself. There should be an option to select
  depth of followed pointer, moreover the code should detect loops and avoid them
* output coloring and formatting, togglable. For example when printing headings they should
  be bold. User should be able to set formatting using tags.

At this moment the code can print structs and allows to specify two kinds of tags:

* `pretty:"sem=title"` -- provide semantics of a field. Currently only `title` is allowed and it
  causes pretty printer to use this field's value as name of the struct as the whole.
* `pretty:"ord=XX"` -- specify order in which fields will be printed. `XX` is an unsigned integer.
  Sorting order is ascending.
