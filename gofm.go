package main

import (
  "log"
  "fmt"
  "time"
  "os"
  ui "github.com/dcorbe/termui-dpc"
  "github.com/dcorbe/termui-dpc/widgets"
  "strings"
)

type renderWidgets struct {
  listLeftChan chan *widgets.List
  listRightChan chan *widgets.List
  paraFnameLeftChan chan *widgets.Paragraph
  paraFnameRightChan chan *widgets.Paragraph
  paraCtrlsChan chan *widgets.Paragraph
}

var isLeft bool = true
var grid *ui.Grid
var listLeftData []string
var listRightData []string
var listLeftDir string
var listRightDir string

func init () {
  fmt.Println("INIT")
  listLeftDir = "/home"
  listRightDir = "/"
  listRightData = getDirListing(listRightDir)
  listLeftData = getDirListing(listLeftDir)
}

func main () {
  if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

  listLeftChan := make(chan *widgets.List)
  listRightChan := make(chan *widgets.List)
  leftRowChan := make(chan int)
  rightRowChan := make(chan int)
  fnameLeftChan := make(chan string)
  fnameRightChan := make(chan string)
  paraFnameLeftChan := make(chan *widgets.Paragraph)
  paraFnameRightChan := make(chan *widgets.Paragraph)
  paraCtrlsChan := make(chan *widgets.Paragraph)
  var widgetChans renderWidgets
  widgetChans.listLeftChan = listLeftChan
  widgetChans.listRightChan = listRightChan
  widgetChans.paraFnameLeftChan = paraFnameLeftChan
  widgetChans.paraFnameRightChan = paraFnameRightChan
  widgetChans.paraCtrlsChan = paraCtrlsChan
	go batchRender(widgetChans)
	go listLeft(listLeftChan, leftRowChan, fnameLeftChan)
	go listRight(listRightChan, rightRowChan, fnameRightChan)
  go fnameLeft(paraFnameLeftChan, fnameLeftChan)
  go fnameRight(paraFnameRightChan, fnameRightChan)
  go ctrls(paraCtrlsChan)

  leftRow, rightRow := 0, 0
  leftRowChan <- leftRow
  rightRowChan <- rightRow

	uiEvents := ui.PollEvents()
  ticker := time.NewTicker(50 * time.Millisecond).C
	for {
		select {
    case <-ticker:
      leftRowChan <-leftRow
      rightRowChan <-rightRow
    case e := <-uiEvents:
			switch e.ID {
        case "q", "Q", "<C-c>":
          return
        case "<Down>":
          var listLeftLength int
          var listRightLength int
          if isLeft {
            for i := 0; i < len(listLeftData); i++ {
              if listLeftData[i] == "" {
                listLeftLength = i
                break
              }
            }
            if leftRow < listLeftLength {
              leftRow++
            }
            if leftRow == listLeftLength {
              leftRow = listLeftLength - 1
            }
            leftRowChan <-leftRow
            rightRowChan <-rightRow
          } else {
            for i := 0; i < len(listRightData); i++ {
              if listRightData[i] == "" {
                listRightLength = i
                break
              }
            }
            if rightRow < listRightLength {
              rightRow++
            }
            if rightRow == listRightLength {
              rightRow = listRightLength - 1
            }
            rightRowChan <-rightRow
            leftRowChan <-leftRow
          }
        case "<Up>":
          if isLeft {
            if leftRow > 0 {
              leftRow--
            }
          } else {
            if rightRow > 0 {
              rightRow--
            }
          }
          leftRowChan <-leftRow
          rightRowChan <-rightRow
        case "<Left>":
          isLeft = true
          leftRowChan <-leftRow
          rightRowChan <-rightRow
        case "<Right>":
          isLeft = false
          leftRowChan <-leftRow
          rightRowChan <-rightRow
        case "t", "T":
          switch isLeft {
          case true:
            leftRow = 0
            leftRowChan <-leftRow
            rightRowChan <-rightRow
          case false:
            rightRow = 0
            rightRowChan <-rightRow
            leftRowChan <-leftRow
          }
        case "b", "B":
          switch isLeft {
          case true:
            for i, s := range listLeftData {
              if s == "" {
                leftRow = i - 1
                break
              }
            }
            leftRowChan <-leftRow
            rightRowChan <-rightRow
          case false:
            for i, s := range listRightData {
              if s == "" {
                rightRow = i - 1
                break
              }
            }
            rightRowChan <-rightRow
            leftRowChan <-leftRow
          }
        case "<Enter>":
          switch isLeft {
          case true:
            if leftRow == 0 {
              fmt.Println("--->",listLeftDir)
              listLeftDir = changeDirUp(listLeftDir)
              if listLeftDir == "/go" {
                fmt.Println("->",listLeftDir)
                os.Exit(1)
              }
              listLeftData = getDirListing(listLeftDir)
            } else {
              if string(listLeftData[leftRow][0]) == "D" {
                if len(listLeftDir) == 1 {
                  listLeftDir = listLeftDir + listLeftData[leftRow][2:]
                } else {
                  listLeftDir = listLeftDir + "/" + listLeftData[leftRow][2:]
                }
                listLeftData = getDirListing(listLeftDir)
                leftRow = 0
              }
            }
          case false:
            if rightRow == 0 {
              listRightDir = changeDirUp(listRightDir)
              listRightData = getDirListing(listRightDir)
            } else {
              if string(listRightData[rightRow][0]) == "D" {
                listRightDir = listRightDir + "/" + listRightData[rightRow][2:]
                listRightData = getDirListing(listRightDir)
                rightRow = 0
              }
            }
          }
          leftRowChan <-leftRow
          rightRowChan <-rightRow
        case "<Resize>":
  				payload := e.Payload.(ui.Resize)
  				grid.SetRect(0, 0, payload.Width, payload.Height)
  				ui.Clear()
          ui.Render(grid)
      }
		}
	}


  return
}

func fnameLeft (pchan chan *widgets.Paragraph, fileNameChan chan string) {
  p := widgets.NewParagraph()
  p.Title = "Selected File/Dir"
  p.TitleStyle.Fg = ui.ColorYellow
  p.SetRect(0, 20, 45, 35)
  p.TextStyle.Fg = ui.ColorWhite
  p.Border = true
  p.BorderStyle.Fg = ui.ColorWhite
  p.WrapText = true
  //ticker := time.NewTicker(50 * time.Millisecond).C

	for {
    name := <-fileNameChan
    if len(listLeftDir) == 1 {
      p.Text = listLeftDir + name[2:]
    } else {
      p.Text = listLeftDir + "/" + name[2:]
    }
		//select {
		//	case <-ticker:
				pchan <-p
		//}
	}
  return
}

func fnameRight (pchan chan *widgets.Paragraph, fileNameChan chan string) {
  p := widgets.NewParagraph()
  p.Title = "Selected File/Dir"
  p.TitleStyle.Fg = ui.ColorYellow
  p.SetRect(0, 20, 45, 23)
  p.TextStyle.Fg = ui.ColorWhite
  p.Border = true
  p.BorderStyle.Fg = ui.ColorWhite
  //ticker := time.NewTicker(50 * time.Millisecond).C
	for {
    name := <-fileNameChan
    p.Text = name
		//select {
		//	case <-ticker:
				pchan <-p
		//}
	}
  return
}

func listLeft (lchan chan *widgets.List, leftRowChan chan int, fileNameChan chan string) {
	l := widgets.NewList()
	l.Title = "Left Directory"
  l.TitleStyle.Fg = ui.ColorYellow
	l.SetRect(0, 0, 45, 20)
	l.TextStyle.Fg = ui.ColorCyan
  //ticker := time.NewTicker(50 * time.Millisecond).C

	for {
    rowCnt := <-leftRowChan
		//select {
		//	case <-ticker:
				l.Rows = listLeftData[rowCnt:]
        switch isLeft {
          case true:
            l.BorderStyle.Fg = ui.ColorGreen
          case false:
            l.BorderStyle.Fg = ui.ColorWhite
        }
        fileNameChan <-listLeftData[rowCnt]
				lchan <- l
		//}
	}
	return
}

func listRight (lchan chan *widgets.List, rightRowChan chan int, fileNameChan chan string) {
  l2 := widgets.NewList()
	l2.Title = "Right Directory"
  l2.TitleStyle.Fg = ui.ColorYellow
	l2.SetRect(46, 0, 91, 20)
  l2.TextStyle.Fg = ui.ColorCyan
	//ticker := time.NewTicker(50 * time.Millisecond).C

	for {
    rowCnt := <-rightRowChan
		//select {
		//	case <-ticker:
				  l2.Rows = listRightData[rowCnt:]
        switch isLeft {
          case false:
            l2.BorderStyle.Fg = ui.ColorGreen
          case true:
            l2.BorderStyle.Fg = ui.ColorWhite
        }
        fileNameChan <-listRightData[rowCnt]
				lchan <- l2
		//}
	}
	return
}

func ctrls (pchan chan *widgets.Paragraph) {
  p := widgets.NewParagraph()
  p.TitleStyle.Fg = ui.ColorYellow
  p.SetRect(25, 24, 69, 30)
  p.TextStyle.Fg = ui.ColorWhite
  p.Border = false
  //p.BorderStyle.Fg = ui.ColorWhite
  p.Text = "Lt/Rt: Switch Sides   Y/N: Confirm Actions\nUp/Dn: Scroll         L/R: L->R or L<-R\nC: Copy   M: Move     D: Delete   Q: Quit\nT: Top  B: Bottom"
  //ticker := time.NewTicker(50 * time.Millisecond).C
	for {
		//select {
	//		case <-ticker:
				pchan <-p
		//}
	}
  return
}

func batchRender (widgetChans renderWidgets) {
  for {
    //select {
    //case <-ticker:
  	l1 := <-widgetChans.listLeftChan
    l2 := <-widgetChans.listRightChan
    p1 := <-widgetChans.paraFnameLeftChan
    p2 := <-widgetChans.paraFnameRightChan
    p3 := <-widgetChans.paraCtrlsChan
    grid = ui.NewGrid()
    grid.Set(
      ui.NewRow(2.0/3,
        ui.NewCol(1.0/2,l1),
        ui.NewCol(1.0/2,l2),
      ),
      ui.NewRow(0.4/3,
        ui.NewCol(1.0/2, p1),
        ui.NewCol(1.0/2, p2),
      ),
      ui.NewRow(0.6/3, p3),
    )
    termWidth, termHeight := ui.TerminalDimensions()
  	grid.SetRect(0, 0, termWidth, termHeight)
		ui.Render(grid)
    //}
	}
}

func getDirListing (dir string) []string {
  files, err := os.ReadDir(dir)
  if err != nil {
    log.Fatal(err)
  }
  listData := make([]string, len(files)+1000)
  listData[0] = ".."
  for i := 0; i < len(files); i++ {
    preStr := "F|"
    if files[i].IsDir() {
      preStr = "D|"
    }
    listData[i+1] = preStr + files[i].Name()
  }
  return listData
}

func changeDirUp (dir string) string {
  dirSlc := strings.Split(dir, "/")
  idx := 0
  for {
    if idx == len(dirSlc) {
      break
    }
    if dirSlc[idx] == "" {
      dirSlc = append(dirSlc[0:idx], dirSlc[idx+1:len(dirSlc)]...)
    } else {
      idx++
    }
  }
  dir = ""
  for i := 0; i < len(dirSlc)-1; i++ {
    dir = dir + "/" + dirSlc[i]
  }
  if dir == "" {
    dir = "/"
  }
  return dir
}
