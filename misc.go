package main

type Person struct {
	Name      string     `pretty:"sem=title,ord=1"`
	Surname   string     `pretty:"ord=2"`
	Age       uint       `pretty:"ord=3"`
	Abilities []Ability  `pretty:"ord=4"`
	Computer  Computer   `pretty:"ord=6"`
	Servers   []Computer `pretty:"ord=5"`
	Friends   []*Person
}

type Computer struct {
	CPU string
}

type Ability struct {
	Name  string `pretty:"sem=title"`
	Level uint
}

func getPerson() Person {
	friendA := Person{
		Name:    "John's",
		Surname: "Friend",
		Age:     33,
		Abilities: []Ability{
			{Name: "Anime", Level: 99},
			{Name: "osu!", Level: 50000},
		},
		Computer: Computer{
			CPU: "???",
		},
		Friends: []*Person{},
	}

	person := Person{
		Name:      "John",
		Surname:   "Doe",
		Age:       31,
		Abilities: getAbilities(),
		Computer: Computer{
			CPU: "Ryzen 7700",
		},
		Servers: []Computer{
			{CPU: "Cortex A76"},
		},
		Friends: []*Person{&friendA},
	}

	friendA.Friends = append(friendA.Friends, &person)

	return person
}

func getAbilities() []Ability {
	return []Ability{
		{"C Programmer", 5},
		{"Go Programmer", 4},
	}
}
