import { vec2, vec3, vec4, mat4 } from 'gl-matrix';

export enum Direction {
  UP = "up",
  DOWN = "down",
  LEFT = "left",
  RIGHT = "right",
  IN = "in",
  OUT = "out",
}

export class Camera {
  private _project: mat4;
  private _inverse: mat4;
  private screenSizePx: vec2;
  private eye: vec3;
  private focal: vec3;
  private perspective: mat4;
  private view: mat4;
  private inversePerspective: mat4;
  private inverseView: mat4;

  private temp1: vec4;
  private temp2: vec4;
  private jogAmplitude: number;

  constructor() {
    this.screenSizePx = vec2.create();
    this._project = mat4.create();
    this._inverse = mat4.create();
    this.inversePerspective = mat4.create();
    this.inverseView = mat4.create();
    this.perspective = mat4.create();
    this.view = mat4.create();
    this.eye = vec3.fromValues(6.5, 5, 5.2);
    this.focal = vec3.create();
    this.temp1 = vec4.create();
    this.temp2 = vec4.create();
    this.jogAmplitude = 0.1;
  }

  snap(position: vec3 | vec2) {
    if (position.length === 3) {
      vec3.copy(this.eye, position);
    } else {
      this.eye[0] = position[0];
      this.eye[1] = position[1];
    }
    this.update();
  }

  dolly(delta: vec3) {
    if (delta[0] === 0 && delta[1] === 0 && delta[2] === 0) {
      return;
    }
    const zCompensation = this.eye[2] * 0.1;
    const xyFactor = 0.01;
    const zFactor = 0.1;
    this.eye[0] -= delta[0] * xyFactor * zCompensation;
    this.eye[1] -= delta[1] * xyFactor * -1 * zCompensation;
    // TODO: zoom on mouse
    this.eye[2] -= delta[2] * zFactor * -1;
    this.update();
  }

  jog(d: Direction): void {
    switch (d) {
      case Direction.UP:
        this.eye[1] += this.jogAmplitude;
        break;
      case Direction.LEFT:
        this.eye[0] -= this.jogAmplitude;
        break;
      case Direction.RIGHT:
        this.eye[0] += this.jogAmplitude;
        break;
      case Direction.DOWN:
        this.eye[1] -= this.jogAmplitude;
        break;
      case Direction.IN:
        this.eye[2] -= this.jogAmplitude;
        break;
      case Direction.OUT:
        this.eye[2] += this.jogAmplitude;
        break;
      default:
        const _: never = d;
        throw new Error(`Unhandeled direction ${d}`);
    }
    this.update();
  }

  setScreenSize(screenSize: vec2) {
    if (vec2.equals(this.screenSizePx, screenSize)) {
      return;
    }
    vec2.copy(this.screenSizePx, screenSize);
    this.update();
  }

  private update() {
    // Cap:
    this.eye[2] = Math.max(0.1, this.eye[2]);

    this.focal[0] = this.eye[0];
    this.focal[1] = this.eye[1];
    const aspect = this.screenSizePx[0] / this.screenSizePx[1];
    mat4.perspectiveZO(this.perspective, Math.PI / 3, aspect, 0.1, 1000);

    const up = vec3.fromValues(0, 1, 0);
    mat4.lookAt(this.view, this.eye, this.focal, up);

    mat4.multiply(this._project, this.perspective, this.view);

    mat4.invert(this.inversePerspective, this.perspective);

    mat4.invert(this.inverseView, this.view);

    mat4.invert(this._inverse, this._project);
  }

  get project(): mat4 {
    return this._project;
  }

  get inverse(): mat4 {
    return this._inverse;
  }

  toWorld(world: vec2, screen: vec2): void {
    // Screen space is:
    // +------> x (px)
    // |
    // |
    // |
    // V
    // y (px)
    //
    // NDC is:
    //
    // (-1, 1)              +y                (1, 1)
    //                       ^
    //                       |
    //               -x <----+---> +x
    //                       |
    //                       V
    // (-1, -1)             -y                 (1, -1)
    //

    const l0 = this.temp1;
    const l = this.temp2;

    l0[0] = (screen[0] / this.screenSizePx[0]) * 2 - 1;
    l0[1] = ((screen[1] / this.screenSizePx[1]) * 2 - 1) * -1;
    l0[2] = 0;
    l0[3] = 1;

    l[0] = 0;
    l[1] = 0;
    l[2] = 1;
    l[3] = 0;

    vec4.transformMat4(l0, l0, this.inverse);
    vec4.transformMat4(l, l, this.inverse);

    vec4.scale(l, l, l0[2] / l[2]);
    vec4.sub(l, l0, l);

    vec4.scale(l, l, 1 / l[3]);

    world[0] = l[0];
    world[1] = l[1];
  }

  toNdcFromScreen(ndc: vec3, screen: vec2): void {
    vec4.zero(this.temp1);
    this.temp1[0] = (screen[0] / this.screenSizePx[0]) * 2 - 1;
    this.temp1[1] = ((screen[1] / this.screenSizePx[1]) * 2 - 1) * -1;
    this.temp1[2] = 0;
    this.temp1[3] = 1;

    ndc[0] = this.temp1[0];
    ndc[1] = this.temp1[1];
    ndc[2] = this.temp1[2];
  }

  toScreen(screen: vec2, world: vec2): void {
    vec4.zero(this.temp1);
    this.temp1[0] = world[0];
    this.temp1[1] = world[1];
    this.temp1[2] = 0;
    this.temp1[3] = 1;

    vec4.transformMat4(this.temp1, this.temp1, this._project);

    this.temp1[0] = this.temp1[0] / this.temp1[3];
    this.temp1[1] = this.temp1[1] / this.temp1[3];

    screen[0] = ((this.temp1[0] + 1) / 2) * this.screenSizePx[0];
    screen[1] = ((-this.temp1[1] + 1) / 2) * this.screenSizePx[1];
  }
}
