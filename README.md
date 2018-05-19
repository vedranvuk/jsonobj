# Description

Package jsonobj implements an intermediate JSON object that translates less code to less speed by constructing an in-memory object that does the type conversion for you and on-the-fly when accessing its' elements.

Elements can be accessed by a textual path in an inuitive way.

# Example

Given a JSON object of the following structure

	{
		"planets": [
			{
				"name": "Saturn",
				"moons": 62
			},
			{
				"name": "Uranus",
				"moons": 27
			}
		]
	}

yuo parse it liek such

	jf, err := jo.NewFromString(TestJSON)

and access values with a path

	var moonz int64
	if err := jf.Get("planets[1].moons", &moonz); err != nil {
		...

you can set, too

	if err := jf.Set("planets[0].name", "Jenifer Lawrence")

Ba-dum, tss.

# License

Use of this source code is governed by a GNU GPLv3 license
