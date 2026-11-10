package web

import "testing"

func TestMapDragWiresNodeDragAndPersistence(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, js,
		"view.pointerDownHit = hitTest(event.clientX, event.clientY)",
		"beginNodeDrag(view.pointerDownHit, event)",
		"cameraX: view.camera.x",
		"const clickTarget = moved || cancelled ? undefined : view.pointerDownHit;",
		"handleNodeClick(clickTarget)",
	)

	start := functionBody(t, js, "beginNodeDrag")
	requireContains(t, start,
		`node.type === "service"`,
		`device.type !== "device"`,
		"view.dragNodeId = device.id",
		"view.nodeDragStart = {",
	)
	move := functionBody(t, js, "moveDeviceTo")
	requireContains(t, move,
		`child.type === "service"`,
		"child.parentId === node.id",
		"child.x += dx",
		"child.y += dy",
	)
	update := functionBody(t, js, "updateNodeDrag")
	requireContains(t, update, "DRAG_THRESHOLD", "view.dragMoved = true", "dx / view.camera.zoom")
	save := functionBody(t, js, "saveDevicePosition")
	requireContains(t, save, "/position", `method: "POST"`, "JSON.stringify({ x, y })", "pinned: true")
	end := functionBody(t, js, "endNodeDrag")
	requireContains(t, end, `event.type === "pointercancel"`, "saveDevicePosition(node)")
}

func TestPinnedDevicesRestoreAndShowPinAffordance(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	apply := functionBody(t, js, "applySnapshot")
	requireContains(t, apply,
		"device.pinned ? savedX",
		"device.pinned ? savedY",
		"device.pinned && node.id !== view.dragNodeId",
		"moveDeviceTo(node, savedX, savedY)",
	)
	physics := functionBody(t, js, "physicsStep")
	requireContains(t, physics,
		"deviceAnchored(a)",
		"deviceAnchored(b)",
		"deviceAnchored(device)",
		`node.type === "device" && deviceAnchored(node)`,
	)
	draw := functionBody(t, js, "drawDevice")
	requireContains(t, draw, "device.pinned", "drawPinAffordance(point, radius, historicAlpha)")
	pin := functionBody(t, js, "drawPinAffordance")
	requireContains(t, pin, "ctx.arc(", `rgba("#ffd166"`)

	legend := functionBody(t, js, "renderLegend")
	requireContains(t, legend, `class="legend-pin"`, ">pinned<")
	css := readAccessibilityFile(t, "style.css")
	requireContains(t, cssRule(t, css, ".legend-pin {"), "border: 1.5px solid var(--theme-warning);")
}
