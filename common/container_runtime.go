/*
 * Copyright Morpheo Org. 2017
 * 
 * contact@morpheo.co
 * 
 * This software is part of the Morpheo project, an open-source machine
 * learning platform.
 * 
 * This software is governed by the CeCILL license, compatible with the
 * GNU GPL, under French law and abiding by the rules of distribution of
 * free software. You can  use, modify and/ or redistribute the software
 * under the terms of the CeCILL license as circulated by CEA, CNRS and
 * INRIA at the following URL "http://www.cecill.info".
 * 
 * As a counterpart to the access to the source code and  rights to copy,
 * modify and redistribute granted by the license, users are provided only
 * with a limited warranty  and the software's author,  the holder of the
 * economic rights,  and the successive licensors  have only  limited
 * liability.
 * 
 * In this respect, the user's attention is drawn to the risks associated
 * with loading,  using,  modifying and/or developing or reproducing the
 * software by the user in light of its specific status of free software,
 * that may mean  that it is complicated to manipulate,  and  that  also
 * therefore means  that it is reserved for developers  and  experienced
 * professionals having in-depth computer knowledge. Users are therefore
 * encouraged to load and test the software's suitability as regards their
 * requirements in conditions enabling the security of their systems and/or
 * data to be ensured and,  more generally, to use and operate it in the
 * same conditions as regards security.
 * 
 * The fact that you are presently reading this means that you have had
 * knowledge of the CeCILL license and that you accept its terms.
 */

package common

import "io"

// ContainerRuntime abstracts Docker/rkt/... it can load/unload images and run them, in a secured
// way :)
type ContainerRuntime interface {
	// ImageBuildAndLoad builds an Image from a reader on a tar.gz archive containing all requirements
	// to build the image. It returns an io.ReadCloser on the image and an error if error there is.
	ImageBuild(name string, buildContext io.Reader) (image io.ReadCloser, err error)

	// ImageLoad loads a saved image from an io.Reader into the container runtime
	ImageLoad(name string, imageReader io.Reader) error

	// ImageUnload removes an Image from the ContainerRuntime's image store (aka from disk)
	ImageUnload(name string) error

	// Runs a given command in a network isolated container
	RunImageInUntrustedContainer(imageName string, args []string, mounts map[string]string, autoRemove bool) (containerID string, err error)

	// SnapshotContainer gets a snapshot of a given container and returns a ReadCloser on it.
	//
	// Note that it is up to the caller to call Close on the returned ReadCloser
	SnapshotContainer(containerID, imageName string) (image io.ReadCloser, err error)
}
