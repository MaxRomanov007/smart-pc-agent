//go:build darwin

package mediactl

/*
#cgo LDFLAGS: -framework CoreFoundation -framework CoreGraphics -framework AppKit

#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <AppKit/NSEvent.h>

// NX media key codes (from IOKit/hidsystem/ev_keymap.h)
#define NX_KEYTYPE_PLAY        16
#define NX_KEYTYPE_NEXT        17
#define NX_KEYTYPE_PREVIOUS    18

// Send a synthetic NX system-defined event (media key)
static void sendMediaKey(int keyCode) {
    // Key down
    NSEvent *downEvent = [NSEvent otherEventWithType:NSEventTypeSystemDefined
                                           location:NSZeroPoint
                                      modifierFlags:0xa00
                                          timestamp:0
                                       windowNumber:0
                                            context:nil
                                            subtype:8      // NX_SUBTYPE_AUX_CONTROL_BUTTONS
                                              data1:(keyCode << 16) | (0xa << 8)  // key down flags
                                              data2:-1];
    CGEventRef cgDown = [downEvent CGEvent];
    CGEventPost(kCGSessionEventTap, cgDown);

    // Key up
    NSEvent *upEvent = [NSEvent otherEventWithType:NSEventTypeSystemDefined
                                         location:NSZeroPoint
                                    modifierFlags:0xb00
                                        timestamp:0
                                     windowNumber:0
                                          context:nil
                                          subtype:8
                                            data1:(keyCode << 16) | (0xb << 8) | 1  // key up flags
                                            data2:-1];
    CGEventRef cgUp = [upEvent CGEvent];
    CGEventPost(kCGSessionEventTap, cgUp);
}
*/
import "C"
import "fmt"

func sendKey(action Action) error {
	switch action {
	case Play:
		C.sendMediaKey(C.NX_KEYTYPE_PLAY)
	case Next:
		C.sendMediaKey(C.NX_KEYTYPE_NEXT)
	case Prev:
		C.sendMediaKey(C.NX_KEYTYPE_PREVIOUS)
	default:
		return fmt.Errorf("mediactl: unknown action %d", action)
	}
	return nil
}
