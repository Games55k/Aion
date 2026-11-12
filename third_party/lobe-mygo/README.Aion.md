This is a local copy of github.com/lobehub/lobe-mygo, used through the
replace directive in the module at the repository root.

The only Aion-specific change is in pkg/field/datepicker.go: the AllowClear
button is stretched to the field height (`Top(0).Bottom(0)`) and its icon is
centered, so it no longer sits at the top of the date field (issue #1).
