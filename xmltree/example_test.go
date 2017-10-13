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

func ExampleUnmarshal() {
	var input = []byte(`<mediawiki xml:lang="en">
	  <page>
	    <title>Page title</title>
	    <restrictions>edit=sysop:move=sysop</restrictions>
	    <revision>
	      <timestamp>2001-01-15T13:15:00Z</timestamp>
	      <contributor><username>Foobar</username></contributor>
	      <comment>I have just one thing to say!</comment>
	      <text>A bunch of [[text]] here.</text>
	      <minor />
	    </revision>
	    <revision>
	      <timestamp>2001-01-15T13:10:27Z</timestamp>
	      <contributor><ip>10.0.0.2</ip></contributor>
	      <comment>new!</comment>
	      <text>An earlier [[revision]].</text>
	    </revision>
	  </page>
	  
	  <page>
	    <title>Talk:Page title</title>
	    <revision>
	      <timestamp>2001-01-15T14:03:00Z</timestamp>
	      <contributor><ip>10.0.0.2</ip></contributor>
	      <comment>hey</comment>
	      <text>WHYD YOU LOCK PAGE??!!! i was editing that jerk</text>
	    </revision>
	  </page>
	</mediawiki>`)

	type revision struct {
		Timestamp   string   `xml:"timestamp"`
		Contributor string   `xml:"contributor>ip"`
		Comment     string   `xml:"comment"`
		Text        []string `xml:"text"`
	}

	root, err := xmltree.Parse(input)
	if err != nil {
		log.Fatal(err)
	}

	// Pull all <revision> items from the input
	for _, el := range root.Search("", "revision") {
		var rev revision
		if err := xmltree.Unmarshal(el, &rev); err != nil {
			log.Print(err)
			continue
		}
		fmt.Println(rev.Timestamp, rev.Comment)
	}

	// Output:
	// 2001-01-15T13:15:00Z I have just one thing to say!
	// 2001-01-15T13:10:27Z new!
	// 2001-01-15T14:03:00Z hey
}

func ExampleMarshal() {
	var input = []byte(`<?xml version="1.0" encoding="UTF-8"?>
	<toc>
	  <chapter-list>
	    <chapter>
	      <title>Civilizing Huck.Miss Watson.Tom Sawyer Waits.</title>
	      <number>1</number>
	    </chapter>
	    <chapter>
	      <title>The Boys Escape Jim.Torn Sawyer's Gang.Deep-laid Plans.</title>
	      <number>2</number>
	    </chapter>
	    <chapter>
	      <title>A Good Going-over.Grace Triumphant."One of Tom Sawyers's Lies".</title>
	      <number>3</number>
	    </chapter>
	    <chapter>
	      <title>Huck and the Judge.Superstition.</title>
	      <number>4</number>
	    </chapter>
	  </chapter-list>
	</toc>`)

	var chapters []xmltree.Element
	root, err := xmltree.Parse(input)
	if err != nil {
		log.Fatal(err)
	}

	for _, el := range root.Search("", "chapter") {
		for _, child := range el.Search("", "title") {
			el.Content = child.Content
		}
		el.Children = nil
		chapters = append(chapters, *el)
	}
	root.Children = chapters
	fmt.Printf("%s\n", xmltree.MarshalIndent(root, "", "  "))

	// Output:
	// <toc>
	//   <chapter>
	//     Civilizing Huck.Miss Watson.Tom Sawyer Waits.
	//   </chapter>
	//   <chapter>
	//     The Boys Escape Jim.Torn Sawyer's Gang.Deep-laid Plans.
	//   </chapter>
	//   <chapter>
	//     A Good Going-over.Grace Triumphant."One of Tom Sawyers's Lies".
	//   </chapter>
	//   <chapter>
	//     Huck and the Judge.Superstition.
	//   </chapter>
	// </toc>
}
