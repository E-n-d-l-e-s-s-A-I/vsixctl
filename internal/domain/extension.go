package domain

type ExtensionID struct {
    Publisher string
    Name      string
}

type Extension struct {
    ID      ExtensionID
    Version string
}