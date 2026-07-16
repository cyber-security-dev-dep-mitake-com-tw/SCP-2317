import QtQuick 2.15
import QtQuick.Controls 2.15
import QtQuick.Layouts 1.15
import Qt.labs.platform 1.1 as Platform
import org.julialang 1.0

// The window starts hidden. Only the tray icon is visible until the user
// clicks it, matching the "silent unless the tray is clicked" requirement.
ApplicationWindow {
    id: root
    visible: false
    width: 560
    height: 680
    minimumWidth: 460
    minimumHeight: 520
    title: "SCP-2317 Companion"
    color: "#0f1419"

    property string healthStatus: "…"
    property var stats: ({})
    property var events: []

    function refresh() {
        root.healthStatus = Julia.fetch_health()
        try { root.stats = JSON.parse(Julia.fetch_stats()) } catch (e) { root.stats = ({}) }
        try { root.events = JSON.parse(Julia.fetch_events()) } catch (e) { root.events = [] }
        eventModel.clear()
        for (var i = 0; i < root.events.length; i++) {
            eventModel.append(root.events[i])
        }
    }

    function popOut() {
        root.show()
        root.raise()
        root.requestActivate()
        refresh()
    }

    Timer {
        interval: 5000
        running: root.visible
        repeat: true
        onTriggered: refresh()
    }

    Component.onCompleted: refresh()

    // ---- System tray ----
    Platform.SystemTrayIcon {
        id: tray
        visible: true
        tooltip: "SCP-2317 Email Protection"
        icon.source: Qt.resolvedUrl("tray.png")
        onActivated: function(reason) {
            if (reason === Platform.SystemTrayIcon.Trigger ||
                reason === Platform.SystemTrayIcon.DoubleClick) {
                root.popOut()
            }
        }
        menu: Platform.Menu {
            Platform.MenuItem { text: "Open dashboard"; onTriggered: root.popOut() }
            Platform.MenuItem { separator: true }
            Platform.MenuItem { text: "Quit"; onTriggered: Qt.quit() }
        }
    }

    // Closing the window hides it back to the tray instead of quitting.
    onClosing: function(close) {
        close.accepted = false
        root.hide()
    }

    // ---- Dashboard ----
    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 18
        spacing: 16

        RowLayout {
            Layout.fillWidth: true
            Text { text: "SCP-2317"; color: "#e6edf3"; font.pixelSize: 24; font.bold: true }
            Item { Layout.fillWidth: true }
            Rectangle {
                width: 96; height: 28; radius: 14
                color: root.healthStatus === "ok" ? "#132e1a" : "#3a1414"
                border.color: root.healthStatus === "ok" ? "#2ea043" : "#f85149"
                Text {
                    anchors.centerIn: parent
                    text: root.healthStatus === "ok" ? "● service" : "● offline"
                    color: root.healthStatus === "ok" ? "#3fb950" : "#f85149"
                    font.pixelSize: 12
                }
            }
        }

        // Stat tiles
        RowLayout {
            Layout.fillWidth: true
            spacing: 12
            Repeater {
                model: [
                    { key: "malicious", label: "Malicious", accent: "#f85149" },
                    { key: "suspicious", label: "Suspicious", accent: "#d29922" },
                    { key: "clean", label: "Clean", accent: "#3fb950" }
                ]
                delegate: Rectangle {
                    Layout.fillWidth: true
                    height: 84
                    radius: 10
                    color: "#161b22"
                    border.color: "#30363d"
                    ColumnLayout {
                        anchors.centerIn: parent
                        spacing: 2
                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: (root.stats.by_status && root.stats.by_status[modelData.key]) ? root.stats.by_status[modelData.key] : "0"
                            color: modelData.accent
                            font.pixelSize: 28; font.bold: true
                        }
                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: modelData.label; color: "#8b949e"; font.pixelSize: 12
                        }
                    }
                }
            }
        }

        Text {
            text: "Total events: " + (root.stats.total ? root.stats.total : 0)
            color: "#8b949e"; font.pixelSize: 12
        }

        // Recent events
        Text { text: "Recent activity"; color: "#e6edf3"; font.pixelSize: 15; font.bold: true }
        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            radius: 10
            color: "#161b22"
            border.color: "#30363d"
            ListView {
                id: eventList
                anchors.fill: parent
                anchors.margins: 8
                clip: true
                spacing: 4
                model: ListModel { id: eventModel }
                delegate: RowLayout {
                    width: eventList.width
                    spacing: 8
                    Rectangle {
                        width: 8; height: 8; radius: 4
                        Layout.alignment: Qt.AlignVCenter
                        color: status === "malicious" ? "#f85149" :
                               status === "suspicious" ? "#d29922" : "#3fb950"
                    }
                    Text {
                        text: (kind ? kind : "event") + ": " + (summary ? summary : "")
                        color: "#c9d1d9"; font.pixelSize: 12
                        elide: Text.ElideRight
                        Layout.fillWidth: true
                    }
                    Text {
                        text: score ? ("" + score) : ""
                        color: "#8b949e"; font.pixelSize: 12
                    }
                }
            }
        }

        Button {
            Layout.fillWidth: true
            text: "Refresh now"
            onClicked: root.refresh()
        }
    }
}
