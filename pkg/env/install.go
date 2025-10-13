package env

// InstallMethod is the installation method (manifest, helm, rhacs-operator).
var InstallMethod = RegisterSetting("ROX_INSTALL_METHOD", AllowEmpty())
