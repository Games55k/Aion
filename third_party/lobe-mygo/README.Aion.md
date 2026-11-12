This is a local copy of github.com/lobehub/lobe-mygo, used through the
replace directive in the module at the repository root.

Aion-specific changes:

- `pkg/field/datepicker.go`: the AllowClear button is stretched to the field
  height (`Top(0).Bottom(0)`) and its icon is centered, so it no longer sits at
  the top of the date field (issue #1).
- `pkg/display/hotkey.go` and `pkg/field/searchbar.go`: added `theme.Small`
  sizing for `display.Hotkey` and passed `SearchBarProps.Size` so shortcut tags
  inside compact (`Small`) search bars have vertical and right-side breathing
  room.
- `pkg/action/button.go`: added `Danger bool` to `ButtonProps` so destructive
  buttons render with theme error borders, fills, and text across variants.
