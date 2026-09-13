import QtQuick
import QtQuick.Controls
import Quickshell.Io
import qs.Ui
import qs.Commons

// Downloads bar widget for the dm daemon.
//
// A long-lived `dm watch` process streams JSON snapshots (one per line);
// launching it also auto-starts the daemon. The widget therefore never
// polls — it renders whatever the last snapshot said. If the process dies
// (daemon killed, binary missing) a one-shot timer restarts it after a
// pause, so the shell never spins.
Panel {
  id: root
  moduleName: "shahriyar.downloads"
  ipcTarget: "shahriyar.downloads"

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  // Last snapshot from `dm watch`: { active, totalSpeed, jobs, recent }
  property var snapshot: ({ active: 0, totalSpeed: 0, jobs: [], recent: [] })
  property bool daemonUp: false

  readonly property var liveJobs: (snapshot && snapshot.jobs) ? snapshot.jobs : []
  readonly property var recentJobs: (snapshot && snapshot.recent) ? snapshot.recent : []
  readonly property int activeCount: (snapshot && snapshot.active) ? snapshot.active : 0

  function human(n) {
    n = Number(n) || 0
    if (n >= 1073741824) return (n / 1073741824).toFixed(1) + "GB"
    if (n >= 1048576) return (n / 1048576).toFixed(1) + "MB"
    if (n >= 1024) return (n / 1024).toFixed(0) + "KB"
    return n + "B"
  }

  function overallPercent() {
    var done = 0, total = 0
    for (var i = 0; i < liveJobs.length; i++) {
      var j = liveJobs[i]
      if (j.state === "active" && j.total > 0) { done += j.done; total += j.total }
    }
    if (total <= 0) return -1
    return Math.floor(done * 100 / total)
  }

  function barText() {
    if (activeCount === 0) return ""
    var t = " " + activeCount + "  " + human(snapshot.totalSpeed) + "/s"
    var pct = overallPercent()
    if (pct >= 0) t += "  " + pct + "%"
    return t
  }

  function jobPercent(j) {
    if (!j || j.total <= 0) return 0
    return Math.min(1, j.done / j.total)
  }

  function jobCaption(j) {
    if (j.state === "active") {
      var s = human(j.done) + " / " + human(j.total > 0 ? j.total : j.done) + " · " + human(j.speed) + "/s"
      if (j.eta >= 0) s += " · " + (j.eta > 90 ? Math.ceil(j.eta / 60) + "m" : j.eta + "s") + " left"
      return s
    }
    if (j.state === "failed") return "failed" + (j.error ? ": " + j.error : "")
    if (j.state === "done") return human(j.total)
    return j.state
  }

  function openFolder(j) {
    // Direct argv, no shell: directory paths need no quoting this way.
    var p = actProc.createObject(root, { command: ["xdg-open", j.dir] })
    if (p) p.running = true
  }

  function act(verb, id) {
    // Login shell so ~/.local/bin/dm is found regardless of the shell's PATH.
    var p = actProc.createObject(root, { command: ["bash", "-lc", "dm " + verb + " " + id] })
    if (p) p.running = true
  }

  Component {
    id: actProc
    Process {
      onExited: Qt.callLater(function() { destroy() })
    }
  }

  // Reloads and rescans destroy the widget; without this the spawned
  // `dm watch` would outlive it and pile up.
  Component.onDestruction: {
    restartTimer.stop()
    watchProc.running = false
  }

  Process {
    id: watchProc
    command: ["bash", "-lc", "exec dm watch"]
    running: true
    stdout: SplitParser {
      onRead: function(line) {
        try {
          var s = JSON.parse(line)
          if (s && s.type === "snapshot") {
            root.snapshot = s
            root.daemonUp = true
          }
        } catch (e) { /* partial or garbage line: keep last snapshot */ }
      }
    }
    onExited: {
      root.daemonUp = false
      root.snapshot = { active: 0, totalSpeed: 0, jobs: [], recent: root.recentJobs }
      restartTimer.restart()
    }
  }

  Timer {
    id: restartTimer
    interval: 3000
    repeat: false
    onTriggered: watchProc.running = true
  }

  WidgetButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: root.barText()
    fontSize: Style.font.bodySmall
    dimmed: root.activeCount === 0
    useActiveColor: false
    tooltipText: root.activeCount > 0
      ? root.activeCount + " downloading · " + root.human(root.snapshot.totalSpeed) + "/s"
      : "Downloads"

    onPressed: function(b) {
      if (root.opened) root.close()
      else root.open()
    }
  }

  KeyboardPanel {
    id: panel
    anchorItem: button
    owner: root
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: panel.fittedContentWidth(Style.space(360))
    contentHeight: panel.fittedContentHeight(content.implicitHeight, Style.space(520))

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      onCloseRequested: root.close()
      onTabRequested: function(direction) { root.switchPanel(direction) }

      Column {
        id: content
        anchors.fill: parent
        spacing: Style.space(10)

        // ---------- Header ----------
        Column {
          width: parent.width
          spacing: Style.space(2)

          Text {
            text: "Downloads"
            color: Color.foreground
            font.family: Style.font.family
            font.pixelSize: Style.font.body
            font.bold: true
          }

          Text {
            text: {
              if (!root.daemonUp) return "daemon starting…"
              if (root.activeCount > 0) return root.activeCount + " active · " + root.human(root.snapshot.totalSpeed) + "/s"
              if (root.liveJobs.length > 0) return root.liveJobs.length + " waiting"
              return "idle"
            }
            color: Color.foreground
            opacity: 0.55
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
          }
        }

        PanelSeparator { width: parent.width }

        Text {
          width: parent.width
          visible: root.liveJobs.length === 0 && root.recentJobs.length === 0
          text: "Nothing yet — downloads from Chrome land here."
          wrapMode: Text.WordWrap
          color: Color.foreground
          opacity: 0.45
          font.family: Style.font.family
          font.pixelSize: Style.font.caption
        }

        // ---------- Live jobs ----------
        Column {
          width: parent.width
          spacing: Style.space(6)
          visible: root.liveJobs.length > 0

          Repeater {
            model: root.liveJobs

            Item {
              id: liveRow
              required property var modelData
              width: parent.width
              implicitHeight: liveCol.implicitHeight + Style.space(8)

              Rectangle {
                anchors.fill: parent
                radius: Style.cornerRadius > 0 ? Style.space(6) : 0
                color: liveHover.containsMouse
                  ? Style.selectedFillFor(Color.foreground, Color.accent)
                  : "transparent"
              }

              Column {
                id: liveCol
                anchors.left: parent.left
                anchors.right: liveActions.left
                anchors.leftMargin: Style.space(6)
                anchors.rightMargin: Style.space(8)
                anchors.verticalCenter: parent.verticalCenter
                spacing: Style.space(3)

                Text {
                  width: parent.width
                  text: liveRow.modelData.name
                  elide: Text.ElideMiddle
                  color: Color.foreground
                  font.family: Style.font.family
                  font.pixelSize: Style.font.bodySmall
                }

                Rectangle {
                  width: parent.width
                  height: Style.space(4)
                  radius: height / 2
                  color: Qt.alpha(Color.foreground, 0.15)

                  Rectangle {
                    width: parent.width * root.jobPercent(liveRow.modelData)
                    height: parent.height
                    radius: parent.radius
                    color: liveRow.modelData.state === "failed" ? Color.urgent : Color.accent
                  }
                }

                Text {
                  width: parent.width
                  text: root.jobCaption(liveRow.modelData)
                  elide: Text.ElideRight
                  color: liveRow.modelData.state === "failed" ? Color.urgent : Color.foreground
                  opacity: liveRow.modelData.state === "failed" ? 0.9 : 0.45
                  font.family: Style.font.family
                  font.pixelSize: Style.font.caption
                }
              }

              Row {
                id: liveActions
                anchors.right: parent.right
                anchors.rightMargin: Style.space(6)
                anchors.verticalCenter: parent.verticalCenter
                spacing: Style.space(4)

                PanelActionButton {
                  iconText: liveRow.modelData.state === "active" ? "" : ""
                  tooltipText: liveRow.modelData.state === "active" ? "Pause" : "Resume"
                  foreground: Color.foreground
                  visible: ["active", "paused", "failed", "queued"].indexOf(liveRow.modelData.state) >= 0
                  onClicked: root.act(liveRow.modelData.state === "active" ? "pause" : "resume",
                                      liveRow.modelData.id)
                }

                PanelActionButton {
                  iconText: ""
                  tooltipText: "Cancel"
                  foreground: Color.foreground
                  onClicked: root.act("cancel", liveRow.modelData.id)
                }
              }

              MouseArea {
                id: liveHover
                anchors.fill: parent
                hoverEnabled: true
                acceptedButtons: Qt.NoButton
              }
            }
          }
        }

        PanelSeparator {
          width: parent.width
          visible: root.liveJobs.length > 0 && root.recentJobs.length > 0
        }

        // ---------- Recent ----------
        Column {
          width: parent.width
          spacing: Style.space(2)
          visible: root.recentJobs.length > 0

          PanelSectionHeader {
            text: "Recent"
            foreground: Color.foreground
          }

          Repeater {
            model: root.recentJobs

            Item {
              id: doneRow
              required property var modelData
              width: parent.width
              implicitHeight: doneText.implicitHeight + Style.space(8)

              Rectangle {
                anchors.fill: parent
                radius: Style.cornerRadius > 0 ? Style.space(6) : 0
                color: doneClick.containsMouse
                  ? Style.selectedFillFor(Color.foreground, Color.accent)
                  : "transparent"
              }

              Column {
                id: doneText
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.leftMargin: Style.space(6)
                anchors.rightMargin: Style.space(6)
                anchors.verticalCenter: parent.verticalCenter
                spacing: Style.space(1)

                Text {
                  width: parent.width
                  text: doneRow.modelData.name
                  elide: Text.ElideMiddle
                  color: Color.foreground
                  font.family: Style.font.family
                  font.pixelSize: Style.font.bodySmall
                }

                Text {
                  width: parent.width
                  text: root.human(doneRow.modelData.total) + " · click to open folder"
                  elide: Text.ElideRight
                  color: Color.foreground
                  opacity: 0.4
                  font.family: Style.font.family
                  font.pixelSize: Style.font.caption
                }
              }

              MouseArea {
                id: doneClick
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: root.openFolder(doneRow.modelData)
              }
            }
          }
        }
      }
    }
  }
}
