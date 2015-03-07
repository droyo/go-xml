package xmltree_test

import (
	"fmt"
	"log"

	"aqwari.net/xml/xmltree"
)

func ExampleElement_Search() {
	data := `
	  <Staff>
        <Person>
            <FullName>Ira Glass</FullName>
        </Person>
        <Person>
            <FullName>Tom Magliozzi</FullName>
        </Person>
        <Person>
            <FullName>Terry Gross</FullName>
        </Person>
    </Staff>
	`
	root, err := xmltree.Parse([]byte(data))
	if err != nil {
		log.Fatal(err)
	}
	for _, el := range root.Search("", "FullName") {
		fmt.Printf("%s\n", el.Content)
	}

	// Output:
	// Ira Glass
	// Tom Magliozzi
	// Terry Gross
}

func ExampleElement_Resolve() {
	data := `
    <collection xmlns:ns="http://ns1.net/">
      <record xmlns:ns="http://ns2.net/">
        <name>Old Town</name>
        <artist xmlns:ns="http://ns3.net/">
          <name>Mustafa Grits</name>
        </artist>
      </record>
      <record xmlns:ns="http://ns4.net/">
        <name>New Town</name>
      </record>
    </collection>
	`
	root, err := xmltree.Parse([]byte(data))
	if err != nil {
		log.Fatal(err)
	}

	// Flatten the tree in depth-first order
	flattened := root.SearchFunc(func(*xmltree.Element) bool { return true })

	fmt.Printf("%s <%s>\n", root.Resolve("ns:foo").Space, root.Name.Local)
	for _, el := range flattened {
		fmt.Printf("%s <%s>\n", el.Resolve("ns:foo").Space, el.Name.Local)
	}

	// Output:
	// http://ns1.net/ <collection>
	// http://ns2.net/ <record>
	// http://ns2.net/ <name>
	// http://ns3.net/ <artist>
	// http://ns3.net/ <name>
	// http://ns4.net/ <record>
	// http://ns4.net/ <name>
}

func ExampleElement_SearchFunc() {
	data := `
	  <People>
        <Person>
            <FullName>Grace R. Emlin</FullName>
            <Email where="home">
                <Addr>gre@example.com</Addr>
            </Email>
            <Email where='work'>
                <Addr>gre@work.com</Addr>
            </Email>
        </Person>
        <Person>
            <FullName>Michael P. Thompson</FullName>
            <Email where="home">
                <Addr>michaelp@example.com</Addr>
            </Email>
            <Email where='work'>
                <Addr>michaelp@work.com</Addr>
                <Addr>michael.thompson@work.com</Addr>
            </Email>
        </Person>
    </People>
	`

	root, err := xmltree.Parse([]byte(data))
	if err != nil {
		log.Fatal(err)
	}

	workEmails := root.SearchFunc(func(el *xmltree.Element) bool {
		return el.Name.Local == "Email" && el.Attr("", "where") == "work"
	})

	for _, el := range workEmails {
		for _, addr := range el.Children {
			fmt.Printf("%s\n", addr.Content)
		}
	}

	// Output:
	// gre@work.com
	// michaelp@work.com
	// michael.thompson@work.com
}
