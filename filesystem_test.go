package sivafs

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/polyfill"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/test"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type CompleteFilesystemSuite struct {
	FilesystemSuite
	test.TempFileSuite
	test.ChrootSuite

	FS SivaFS
}

var _ = Suite(&CompleteFilesystemSuite{})

func (s *CompleteFilesystemSuite) SetUpTest(c *C) {
	s.FilesystemSuite.SetUpTest(c)

	fs := osfs.New(c.MkDir())

	f, err := fs.TempFile("", "siva-fs")
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)

	s.FS, err = NewFilesystem(fs, f.Name(), memfs.New())
	c.Assert(err, IsNil)

	s.BasicSuite.FS = s.FS
	s.DirSuite.FS = s.FS
	s.TempFileSuite.FS = s.FS
	s.ChrootSuite.FS = s.FS
}

func (s *CompleteFilesystemSuite) TestTempFileWithPath(c *C) {
	c.Skip("This test case is not valid for the sivaFS case.")
}

func (s *CompleteFilesystemSuite) TestTempFileManyWithUtil(c *C) {
	c.Skip("This test case is not valid for the sivaFS case.")
}

type FilesystemSuite struct {
	BaseSivaFsSuite
	test.BasicSuite
	test.DirSuite

	FS SivaBasicFS
}

var _ = Suite(&FilesystemSuite{})

func (s *FilesystemSuite) SetUpTest(c *C) {
	fs := osfs.New(c.MkDir())

	f, err := fs.TempFile("", "siva-fs")
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)

	s.FS = New(fs, f.Name())
	s.BasicSuite.FS = polyfill.New(s.FS)
	s.DirSuite.FS = polyfill.New(s.FS)
}

func (s *FilesystemSuite) TestSync(c *C) {
	err := s.FS.Sync()
	c.Assert(err, IsNil)
}

func (s *FilesystemSuite) TestSyncWithOpenFile(c *C) {
	err := s.FS.Sync()
	c.Assert(err, IsNil)

	f, err := s.FS.Create("testOne.txt")
	c.Assert(err, IsNil)

	n, err := f.Write([]byte("qux"))
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 3)

	err = s.FS.Sync()
	c.Assert(err, IsNil)

	n, err = f.Write([]byte("bar"))
	c.Assert(err, Equals, os.ErrClosed)
	c.Assert(n, Equals, 0)

	err = f.Close()
	c.Assert(err, IsNil)

	f, err = s.FS.Open("testOne.txt")
	c.Assert(err, IsNil)
	c.Assert(f, NotNil)

	bytes, err := ioutil.ReadAll(f)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, "qux")
}

func (s *FilesystemSuite) TestOpenFileNotSupported(c *C) {
	_, err := s.FS.OpenFile("testFile.txt", os.O_CREATE, 0)
	c.Assert(err, Equals, billy.ErrNotSupported)

	_, err = s.FS.OpenFile("testFile.txt", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0)
	c.Assert(err, Equals, billy.ErrNotSupported)

	_, err = s.FS.OpenFile("testFile.txt", os.O_RDWR, 0)
	c.Assert(err, Equals, billy.ErrNotSupported)
	_, err = s.FS.OpenFile("testFile.txt", os.O_WRONLY, 0)
	c.Assert(err, Equals, billy.ErrNotSupported)
}

func (s *FilesystemSuite) TestFileReadWriteErrors(c *C) {
	f, err := s.FS.Create("testFile.txt")
	c.Assert(err, IsNil)

	_, err = f.Read(nil)
	c.Assert(err, Equals, ErrWriteOnlyFile)

	_, err = f.Seek(0, 0)
	c.Assert(err, Equals, ErrNonSeekableFile)

	fr, ok := f.(io.ReaderAt)
	c.Assert(ok, Equals, true)
	_, err = fr.ReadAt(nil, 0)
	c.Assert(err, Equals, ErrWriteOnlyFile)
}

func (s *FilesystemSuite) TestFileClosedErrors(c *C) {
	f, err := s.FS.Create("testFile.txt")
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)

	_, err = f.Read(nil)
	c.Assert(err, Equals, os.ErrClosed)

	_, err = f.Seek(0, 0)
	c.Assert(err, Equals, os.ErrClosed)

	_, err = f.Write(nil)
	c.Assert(err, Equals, os.ErrClosed)

	fr, ok := f.(io.ReaderAt)
	c.Assert(ok, Equals, true)
	_, err = fr.ReadAt(nil, 0)
	c.Assert(err, Equals, os.ErrClosed)

	err = f.Close()
	c.Assert(err, Equals, os.ErrClosed)
}

func (s *FilesystemSuite) TestFileOperations(c *C) {
	f1, err := s.FS.Create("testOne.txt")
	c.Assert(err, IsNil)

	_, err = s.FS.Create("testTwo.txt")
	c.Assert(err, Equals, ErrFileWriteModeAlreadyOpen)

	err = f1.Close()
	c.Assert(err, IsNil)

	_, err = s.FS.Create("testTree.txt")
	c.Assert(err, IsNil)

	f1, err = s.FS.Open("testOne.txt")
	c.Assert(err, IsNil)
}

func (s *FilesystemSuite) TestReadFs(c *C) {
	testReadFs(c, false)
}

func testReadFs(c *C, readOnly bool) {
	for _, fixture := range fixtures {
		fs := fixture.FS(c, readOnly)
		c.Assert(fs, NotNil)

		testOpenAndRead(c, fixture, fs)
		testReadDir(c, fixture, fs)
		testStat(c, fixture, fs)
		testNested(c, fixture, fs)
	}
}

func (s *FilesystemSuite) TestCapabilities(c *C) {
	f := fixtures[0]
	fs := f.FS(c, false)

	caps := billy.Capabilities(fs)
	expected := billy.ReadCapability |
		billy.WriteCapability |
		billy.SeekCapability

	c.Assert(caps, Equals, expected)
}

func testOpenAndRead(c *C, f *Fixture, fs billy.Filesystem) {
	for _, path := range f.contents {
		s, err := fs.Stat(path)
		if err != nil {
			println("path", path)
		}
		c.Assert(err, IsNil)

		if !s.IsDir() {
			f, err := fs.Open(path)
			c.Assert(err, IsNil)
			c.Assert(f, NotNil)

			read, err := ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Assert(len(read) > 0, Equals, true)

			err = f.Close()
			c.Assert(err, IsNil)
		}
	}

	file, err := fs.Open("NON-EXISTANT")
	c.Assert(file, IsNil)
	c.Assert(err, Equals, os.ErrNotExist)
}

func testReadDir(c *C, f *Fixture, fs billy.Filesystem) {
	for _, dir := range []string{"", ".", "/"} {
		files, err := fs.ReadDir(dir)
		c.Assert(err, IsNil)
		c.Assert(len(files), Equals, len(f.contents))

		// Here we assume that ReadDir returns contents in physical order.
		for idx, fi := range files {
			c.Assert(f.contents[idx], Equals, fi.Name())
		}
	}

	dirLs, err := fs.ReadDir("NON-EXISTANT")
	c.Assert(err, IsNil)
	c.Assert(dirLs, HasLen, 0)
}

func testStat(c *C, f *Fixture, fs billy.Filesystem) {
	for _, path := range f.contents {
		fi, err := fs.Stat(path)
		c.Assert(err, IsNil)
		c.Assert(fi.Name(), Equals, path)
	}

	fi, err := fs.Stat("NON-EXISTANT")
	c.Assert(fi, IsNil)
	c.Assert(err, Equals, os.ErrNotExist)
}

func testNested(c *C, f *Fixture, fs billy.Filesystem) {
	for _, dir := range f.nested {
		c.Assert(fs, NotNil)

		stat, err := fs.Stat(dir.name)
		c.Assert(err, IsNil)
		c.Assert(stat.IsDir(), Equals, dir.dir)

		if stat.IsDir() {
			files, err := fs.ReadDir(dir.name)
			c.Assert(err, IsNil)
			c.Assert(len(files), Equals, len(dir.files))

			for idx, fi := range files {
				c.Assert(fi.Name(), Equals, dir.files[idx])
			}
		}
	}
}

type BaseSivaFsSuite struct{}

func (s *BaseSivaFsSuite) TestRename(c *C) {
	c.Skip("Rename not supported")
}

func (s *BaseSivaFsSuite) TestOpenFileAppend(c *C) {
	c.Skip("O_APPEND not supported")
}

func (s *BaseSivaFsSuite) TestOpenFileNoTruncate(c *C) {
	c.Skip("O_CREATE without O_TRUNC not supported")
}

func (s *BaseSivaFsSuite) TestOpenFileReadWrite(c *C) {
	c.Skip("O_RDWR not supported")
}

func (s *BaseSivaFsSuite) TestSeekToEndAndWrite(c *C) {
	c.Skip("does not support seek on writeable files")
}

func (s *BaseSivaFsSuite) TestReadAtOnReadWrite(c *C) {
	c.Skip("ReadAt not supported on writeable files")
}

func (s *BaseSivaFsSuite) TestMkdirAll(c *C) {
	c.Skip("MkdirAll method does nothing")
}

func (s *BaseSivaFsSuite) TestMkdirAllIdempotent(c *C) {
	c.Skip("MkdirAll method does nothing")
}

func (s *BaseSivaFsSuite) TestMkdirAllNested(c *C) {
	c.Skip("because MkdirAll does nothing, is not possible to check the " +
		"Stat of a directory created with this mehtod")
}

func (s *BaseSivaFsSuite) TestStatDir(c *C) {
	c.Skip("StatDir is not possible because directories do not exists in siva")
}

func (s *BaseSivaFsSuite) TestRenameToDir(c *C) {
	c.Skip("Dir renaming not supported")
}

func (s *BaseSivaFsSuite) TestRenameDir(c *C) {
	c.Skip("Dir renaming not supported")
}

func (s *BaseSivaFsSuite) TestFileNonRead(c *C) {
	c.Skip("Is not possible to write a file and then read it at the same time")
}

func (s *BaseSivaFsSuite) TestFileWrite(c *C) {
	c.Skip("Open method open a file in write only mode")
}

func (s *BaseSivaFsSuite) TestTruncate(c *C) {
	c.Skip("Truncate is not supported")
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = io.Copy(d, s)
	return err
}

type nestedContent struct {
	name  string
	dir   bool
	files []string
}

type Fixture struct {
	name     string
	unsafe   bool
	contents []string
	nested   []nestedContent
}

const fixturesPath = "fixtures"

var fixtures = []*Fixture{
	{
		name: "basic.siva",
		contents: []string{
			"dir",
			"nested_dir",
			"gopher.txt",
			"readme.txt",
			"todo.txt",
		},
		nested: []nestedContent{
			{"dir", true, []string{"winter.txt"}},
			{"nested_dir", true, []string{"dir"}},
			{"nested_dir/dir", true, []string{"nested_file.txt"}},
			{"nested_dir/dir/nested_file.txt", false, nil},
		},
	}, {
		name: "zipslip.siva",
		contents: []string{
			"imoutside",
		},
	}, {
		name:   "zipslip.siva",
		unsafe: true,
		contents: []string{
			"subdir",
		},
	},
}

func (f *Fixture) Path() string {
	return filepath.Join(fixturesPath, f.name)
}

func (f *Fixture) FS(c *C, readOnly bool) billy.Filesystem {
	fs, err := f.FSOffset(c, readOnly, 0)
	c.Assert(err, IsNil)

	return fs
}

func (f *Fixture) FSOffset(
	c *C,
	readOnly bool,
	offset uint64,
) (billy.Filesystem, error) {
	tmp := c.MkDir()

	err := copyFile(f.Path(), filepath.Join(tmp, f.name))
	if err != nil {
		return nil, err
	}

	options := SivaFSOptions{
		UnsafePaths: f.unsafe,
		ReadOnly:    readOnly,
		Offset:      offset,
	}

	fs, err := NewFilesystemWithOptions(osfs.New(tmp), f.name, memfs.New(), options)

	return fs, err
}

type ReadOnlyFilesystemSuite struct {
}

var _ = Suite(&ReadOnlyFilesystemSuite{})

func (s *ReadOnlyFilesystemSuite) TestReadFs(c *C) {
	testReadFs(c, true)
}

func (s *ReadOnlyFilesystemSuite) TestCapabilities(c *C) {
	f := fixtures[0]
	fs := f.FS(c, true)

	caps := billy.Capabilities(fs)
	expected := billy.ReadCapability |
		billy.SeekCapability

	c.Assert(caps, Equals, expected)
}

func (s *ReadOnlyFilesystemSuite) TestWriteFile(c *C) {
	f := fixtures[0]
	fs := f.FS(c, true)

	testFile := "write.txt"

	_, err := fs.Create(testFile)
	c.Assert(err, Equals, ErrReadOnlyFilesystem)

	_, err = fs.Stat(testFile)
	c.Assert(err, Equals, os.ErrNotExist)

	_, err = fs.Create(testFile)
	c.Assert(err, Equals, ErrReadOnlyFilesystem)

	for _, flag := range []int{os.O_CREATE, os.O_WRONLY, os.O_TRUNC} {
		_, err = fs.OpenFile(testFile, flag, 0664)
		c.Assert(err, Equals, ErrReadOnlyFilesystem)
	}

	file, err := fs.Open("gopher.txt")
	c.Assert(err, IsNil)

	_, err = file.Write([]byte{0, 0, 0, 0})
	c.Assert(err, Equals, ErrReadOnlyFile)
}

func (s *ReadOnlyFilesystemSuite) TestDir(c *C) {
	f := fixtures[0]
	fs := f.FS(c, true)

	err := fs.MkdirAll("new_dir", 0775)
	c.Assert(err, Equals, ErrReadOnlyFilesystem)

	err = fs.Remove("gopher.txt")
	c.Assert(err, Equals, ErrReadOnlyFilesystem)

	err = fs.Remove("dir")
	c.Assert(err, Equals, ErrReadOnlyFilesystem)
}

func (s *ReadOnlyFilesystemSuite) TestOffset(c *C) {
	f := fixtures[0]

	_, err := f.FSOffset(c, true, 0)
	c.Assert(err, IsNil)

	_, err = f.FSOffset(c, true, 10)
	c.Assert(err, IsNil)

	_, err = f.FSOffset(c, false, 0)
	c.Assert(err, IsNil)

	_, err = f.FSOffset(c, false, 10)
	c.Assert(err, Equals, ErrOffsetReadWrite)
}
